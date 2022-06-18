package blobs

import (
	"context"
	"fmt"

	"github.com/adrg/xdg"
	"github.com/vito/bass-loop/pkg/cfg"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
)

type Bucket struct {
	*blob.Bucket
}

func Open(config *cfg.Config) (*Bucket, error) {
	var blobs *blob.Bucket
	var err error
	if config.BlobsBucket != "" {
		blobs, err = blob.OpenBucket(context.TODO(), config.BlobsBucket)
	} else {
		localBlobs, err := xdg.DataFile("bass-loop/blobs")
		if err != nil {
			return nil, fmt.Errorf("xdg: %w", err)
		}

		blobs, err = fileblob.OpenBucket(localBlobs, &fileblob.Options{
			CreateDir: true,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("open bucket: %w", err)
	}
	return &Bucket{blobs}, nil
}
