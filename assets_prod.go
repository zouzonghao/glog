//go:build release

package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:templates
var embedTemplatesFS embed.FS

//go:embed all:static
var embedStaticFS embed.FS

func init() {
	log.Println("Running in release mode, using embedded assets.")
	var err error
	templatesFS, err = fs.Sub(embedTemplatesFS, "templates")
	if err != nil {
		log.Fatal("Failed to create sub filesystem for embedded templates:", err)
	}
	staticFS, err = fs.Sub(embedStaticFS, "static")
	if err != nil {
		log.Fatal("Failed to create sub filesystem for embedded static files:", err)
	}
}
