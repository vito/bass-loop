//go:build tools
// +build tools

// Track tool dependencies to prevent 'go mod tidy' from cleaning them up.
package loop

import _ "github.com/livebud/bud"
