//go:build tools
// +build tools

package tools

import (
	_ "github.com/kisielk/errcheck"
	_ "golang.org/x/lint/golint"
	_ "honnef.co/go/tools/cmd/staticcheck"
)
