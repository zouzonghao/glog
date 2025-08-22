//go:build !release

package main

import (
	"log"
	"os"
)

func init() {
	log.Println("Running in debug mode, using live assets from filesystem.")
	templatesFS = os.DirFS("templates")
	staticFS = os.DirFS("static")
}
