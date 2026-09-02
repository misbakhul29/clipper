package web

import (
	_ "embed"
)

// IndexHTML contains the embedded web dashboard single page application.
//
//go:embed static/index.html
var IndexHTML string
