package models

// Code generated by xo. DO NOT EDIT.

import (
	"context"
)

// RunListing represents a row from 'run_listing'.
type RunListing struct {
	ID string `json:"id"` // id
}

// GetRunListings runs a custom query, returning results as RunListing.
func GetRunListings(ctx context.Context, db DB) ([]*RunListing, error) {
	// query
	const sqlstr = `SELECT id FROM runs ORDER BY start_time DESC`
	// run
	logf(sqlstr)
	rows, err := db.QueryContext(ctx, sqlstr)
	if err != nil {
		return nil, logerror(err)
	}
	defer rows.Close()
	// load results
	var res []*RunListing
	for rows.Next() {
		var rl RunListing
		// scan
		if err := rows.Scan(&rl.ID); err != nil {
			return nil, logerror(err)
		}
		res = append(res, &rl)
	}
	if err := rows.Err(); err != nil {
		return nil, logerror(err)
	}
	return res, nil
}