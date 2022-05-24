package main

import (
	"context"
	"database/sql"

	"gocloud.dev/blob"
)

func sshServe(ctx context.Context, db *sql.DB, blobs *blob.Bucket) error {
	config.SSH.DB = db
	config.SSH.Blobs = blobs
	return config.SSH.ListenAndServe(ctx)
}
