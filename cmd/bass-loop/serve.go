package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/adrg/xdg"
	"github.com/opencontainers/go-digest"
	"github.com/vito/bass/pkg/bass"
	"github.com/vito/bass/pkg/cli"
	"github.com/vito/bass/pkg/ioctx"
	"github.com/vito/bass/pkg/zapctx"
	"github.com/vito/progrock"
	"gocloud.dev/blob"
	"gocloud.dev/blob/fileblob"
	"golang.org/x/sync/errgroup"
)

var fancy bool

func init() {
	fancy = os.Getenv("BASS_FANCY_TUI") != ""
}

func serve(ctx context.Context) error {
	db, err := openDB()
	if err != nil {
		return err
	}

	defer db.Close()

	var blobs *blob.Bucket
	if config.BlobsBucket != "" {
		blobs, err = blob.OpenBucket(ctx, config.BlobsBucket)
	} else {
		localBlobs, err := xdg.DataFile("bass-loop/blobs")
		if err != nil {
			return fmt.Errorf("xdg: %w", err)
		}

		blobs, err = fileblob.OpenBucket(localBlobs, &fileblob.Options{
			CreateDir: true,
		})
	}
	if err != nil {
		return fmt.Errorf("open logs bucket: %w", err)
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	statuses, statusW := progrock.Pipe()
	recorder := progrock.NewRecorder(statusW)

	defer recorder.Stop()

	ctx = progrock.RecorderToContext(ctx, recorder)

	if statuses != nil {
		recorder.Display(stop, cli.ProgressUI, os.Stderr, statuses, fancy)
	}

	eg := new(errgroup.Group)
	spawn(ctx, eg, recorder, "http", func(ctx context.Context) error {
		return httpServe(ctx, db, blobs)
	})

	spawn(ctx, eg, recorder, "ssh", func(ctx context.Context) error {
		return sshServe(ctx, db, blobs)
	})

	return eg.Wait()
}

func spawn(ctx context.Context, eg *errgroup.Group, recorder *progrock.Recorder, name string, f func(context.Context) error) {
	vtx := recorder.Vertex(digest.NewDigestFromEncoded("cmd", name), name)
	eg.Go(func() (err error) {
		defer func() { vtx.Done(err) }()

		stderr := vtx.Stderr()

		// wire up logs to vertex
		logger := bass.LoggerTo(stderr)
		ctx = zapctx.ToContext(ctx, logger)

		// wire up stderr for (log), (debug), etc.
		ctx = ioctx.StderrToContext(ctx, stderr)

		err = f(ctx)
		if err != nil {
			cli.WriteError(ctx, err)
			return
		}

		return
	})
}
