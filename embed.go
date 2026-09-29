package main

import (
	"embed"
	"io/fs"
)

//go:embed sablonok/*.html
var templateFS embed.FS

//go:embed assets/*
var assetsFS embed.FS

//go:embed web
var webFS embed.FS

//go:embed demo/tartalom-minta.json demo/Energofish_partner_hirlevel_minta.xlsx
var demoFS embed.FS

func mustSub(f fs.FS, dir string) fs.FS {
	s, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return s
}
