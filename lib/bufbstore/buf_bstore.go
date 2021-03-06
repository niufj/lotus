package bufbstore

import (
	"context"

	block "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	ds "github.com/ipfs/go-datastore"
	bstore "github.com/ipfs/go-ipfs-blockstore"
)

type BufferedBS struct {
	read  bstore.Blockstore
	write bstore.Blockstore
}

func NewBufferedBstore(base bstore.Blockstore) *BufferedBS {
	buf := bstore.NewBlockstore(ds.NewMapDatastore())
	return &BufferedBS{
		read:  base,
		write: buf,
	}
}

var _ (bstore.Blockstore) = &BufferedBS{}

func (bs *BufferedBS) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	a, err := bs.read.AllKeysChan(ctx)
	if err != nil {
		return nil, err
	}

	b, err := bs.write.AllKeysChan(ctx)
	if err != nil {
		return nil, err
	}

	out := make(chan cid.Cid)
	go func() {
		defer close(out)
		for a != nil || b != nil {
			select {
			case val, ok := <-a:
				if !ok {
					a = nil
				} else {
					select {
					case out <- val:
					case <-ctx.Done():
						return
					}
				}
			case val, ok := <-b:
				if !ok {
					b = nil
				} else {
					select {
					case out <- val:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return out, nil
}

func (bs *BufferedBS) DeleteBlock(c cid.Cid) error {
	if err := bs.read.DeleteBlock(c); err != nil {
		return err
	}

	return bs.write.DeleteBlock(c)
}

func (bs *BufferedBS) Get(c cid.Cid) (block.Block, error) {
	if out, err := bs.read.Get(c); err != nil {
		if err != bstore.ErrNotFound {
			return nil, err
		}
	} else {
		return out, nil
	}

	return bs.write.Get(c)
}

func (bs *BufferedBS) GetSize(c cid.Cid) (int, error) {
	panic("nyi")
}

func (bs *BufferedBS) Put(blk block.Block) error {
	return bs.write.Put(blk)
}

func (bs *BufferedBS) Has(c cid.Cid) (bool, error) {
	has, err := bs.read.Has(c)
	if err != nil {
		return false, err
	}
	if has {
		return true, nil
	}

	return bs.write.Has(c)
}

func (bs *BufferedBS) HashOnRead(hor bool) {
	bs.read.HashOnRead(hor)
	bs.write.HashOnRead(hor)
}

func (bs *BufferedBS) PutMany(blks []block.Block) error {
	return bs.write.PutMany(blks)
}

func (bs *BufferedBS) Read() bstore.Blockstore {
	return bs.read
}
