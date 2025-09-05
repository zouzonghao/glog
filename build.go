//go:build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

var (
	m            = minify.New()
	staticDir    = "static"
	staticBakDir = "static.bak"
)

func init() {
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
}

func main() {
	release := flag.Bool("release", false, "Process assets for release")
	clean := flag.Bool("clean", false, "Clean processed assets and restore original files")
	flag.Parse()

	if *release && *clean {
		log.Fatal("Cannot use -release and -clean flags simultaneously.")
	}

	if *release {
		fmt.Println("Processing assets for release...")
		if err := processAssets(); err != nil {
			log.Fatalf("Failed to process assets for release: %v", err)
		}
		fmt.Println("Assets processed successfully.")
	} else if *clean {
		fmt.Println("Cleaning up processed assets...")
		if err := cleanupAssets(); err != nil {
			log.Fatalf("Failed to clean up assets: %v", err)
		}
		fmt.Println("Cleanup complete.")
	} else {
		fmt.Println("No action specified. Use -release to process assets or -clean to clean up.")
	}
}

func processAssets() error {
	// 1. Rename static to static.bak
	if _, err := os.Stat(staticBakDir); err == nil {
		return fmt.Errorf("backup directory '%s' already exists, please run 'make clean' first", staticBakDir)
	}
	if err := os.Rename(staticDir, staticBakDir); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", staticDir, staticBakDir, err)
	}
	fmt.Printf("Renamed '%s' to '%s'\n", staticDir, staticBakDir)

	// 2. Create new static directory
	if err := os.Mkdir(staticDir, 0755); err != nil {
		return err
	}

	// 3. Minify CSS and JS assets
	minifyDirs := []string{"css", "js"}
	for _, dir := range minifyDirs {
		sourceDir := filepath.Join(staticBakDir, dir)
		destDir := filepath.Join(staticDir, dir)

		if err := os.Mkdir(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		if _, err := os.Stat(sourceDir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			// Minify .css and .js files
			if strings.HasSuffix(info.Name(), ".css") || strings.HasSuffix(info.Name(), ".js") {
				destPath := filepath.Join(destDir, info.Name())
				if err := minifyFile(path, destPath); err != nil {
					return fmt.Errorf("failed to minify %s: %w", path, err)
				}
				fmt.Printf("Minified '%s' to '%s'\n", path, destPath)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	// 4. Copy other assets like 'pic'
	picSourceDir := filepath.Join(staticBakDir, "pic")
	if _, err := os.Stat(picSourceDir); err == nil {
		picDestDir := filepath.Join(staticDir, "pic")
		if err := copyDir(picSourceDir, picDestDir); err != nil {
			return fmt.Errorf("failed to copy pic directory: %w", err)
		}
		fmt.Printf("Copied '%s' to '%s'\n", picSourceDir, picDestDir)
	}

	return nil
}

func cleanupAssets() error {
	// 1. Remove the temporary static directory
	if err := os.RemoveAll(staticDir); err != nil {
		return fmt.Errorf("failed to remove temporary '%s': %w", staticDir, err)
	}
	fmt.Printf("Removed temporary '%s'\n", staticDir)

	// 2. Rename static.bak back to static
	if _, err := os.Stat(staticBakDir); os.IsNotExist(err) {
		fmt.Println("No backup directory to restore.")
		return nil
	}
	if err := os.Rename(staticBakDir, staticDir); err != nil {
		return fmt.Errorf("failed to rename '%s' back to '%s': %w", staticBakDir, staticDir, err)
	}
	fmt.Printf("Restored '%s' from '%s'\n", staticDir, staticBakDir)

	return nil
}

func minifyFile(inPath, outPath string) error {
	inFile, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer inFile.Close()

	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var mime string
	switch filepath.Ext(inPath) {
	case ".css":
		mime = "text/css"
	case ".js":
		mime = "application/javascript"
	default:
		// Should not happen based on the Walk logic, but good to have
		return copyFile(inPath, outPath)
	}

	return m.Minify(mime, outFile, inFile)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
