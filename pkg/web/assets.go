package web

import (
	"embed"
)

// StaticFS contains all embedded web dashboard static assets (HTML, CSS, JS).
//
//go:embed static/*
var StaticFS embed.FS
