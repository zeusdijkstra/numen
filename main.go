package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type config struct {
	list bool
	ext  string
	size int64
}

func main() {
	root := flag.String("root", "", "this is the root directory to start")
	list := flag.Bool("list", false, "list flag")
	ext := flag.String("ext", "", "extension flag")
	size := flag.Int64("min", 0, "minimal size of the file")
	flag.Parse()

	c := config{
		list: *list,
		ext:  *ext,
		size: *size,
	}

	if err := run(*root, os.Stdout, c); err != nil {
		fmt.Println("error! from run()")
		os.Exit(1)
	}
}

func run(root string, out io.Writer, cfg config) error {
	fileSystem := os.DirFS(root)
	return fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Early exit for directories using DirEntry (faster)
		if d.IsDir() {
			return nil
		}

		// Only get FileInfo if we need size information
		info, err := d.Info()
		if err != nil {
			return err
		}

		if filterOut(path, cfg.ext, cfg.size, info) {
			return nil
		}

		return listFile(path, out)
	})
}

func filterOut(path, ext string, minSize int64, info fs.FileInfo) bool {
	// We already checked IsDir() above, so just check size and extension
	if info.Size() < minSize {
		return true
	}
	if ext != "" && filepath.Ext(path) != ext {
		return true
	}
	return false
}

func listFile(path string, out io.Writer) error {
	_, err := fmt.Fprintln(out, path)
	return err
}
