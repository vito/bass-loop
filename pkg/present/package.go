// Package present loads state from the database and turns it into properties
// to expose to the frontend.
//
// It leverages SQLite's speed to keep loading simple and return rich data by
// loading it from the database whenever it's needed, even if it's already been
// loaded previously.
package present
