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
	return fs.WalkDir(fileSystem, ".",
		// our callback function
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			if !shouldKeep(path, cfg.ext, cfg.size, info) {
				return nil
			}

			return listFile(path, out)
		})
}

func shouldKeep(path, ext string, minSize int64, info fs.FileInfo) bool {
	if info.Size() < minSize {
		return false
	}
	if ext != "" && filepath.Ext(path) != ext {
		return false
	}
	return true
}

func listFile(path string, out io.Writer) error {
	_, err := fmt.Fprintln(out, path)
	return err
}
