package main

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type config struct {
	list bool
	ext  []string
	size int64
	del  []string
}

func main() {
	root := flag.String("root", "", "this is the root directory to start")
	list := flag.Bool("list", false, "list flag")
	ext := flag.String("ext", "", "extension flag")
	size := flag.Int64("min", 0, "minimal size of the file")
	del := flag.String("del", "", "delete file")
	flag.Parse()

	c := config{
		list: *list,
		ext:  handleMultiple(*ext),
		size: *size,
		del:  handleMultiple(*del),
	}

	if err := run(*root, os.Stdout, c); err != nil {
		fmt.Println("there is an error running the code")
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

			info, err := d.Info()
			if err != nil {
				return err
			}

			if !shouldKeep(path, cfg.ext, cfg.size, info) {
				return nil
			}

			if cfg.del != nil {
				listFile := []string{}
				for _, file := range cfg.del {
					full := filepath.Join(root, file)
					listFile = append(listFile, full)
				}
				err := deleteFile(listFile)
				if err != nil {
					return err
				}
				fmt.Println(msg)
				return nil
			}

			// list only if we are not deleting file
			if cfg.del == nil {
				return listFile(path, out)
			}

			return nil
		})
}

// ACTIONS
func shouldKeep(path string, exts []string, minSize int64, info fs.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	if info.Size() < minSize {
		return false
	}

	// keep all files if there is no extensions
	if len(exts) == 0 || (len(exts) == 1 && exts[0] == "") {
		return true
	}

	fileExt := filepath.Ext(path)
	return slices.Contains(exts, fileExt)
}

func listFile(path string, out io.Writer) error {
	_, err := fmt.Fprintln(out, path)
	return err
}

func deleteFile(files []string) error {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
		fmt.Sprintf("sucesfully deleted: %s", file)
	}
	return nil
}

func handleMultiple(input string) []string {
	exts := strings.Fields(input)
	return exts
}
