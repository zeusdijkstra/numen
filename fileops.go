package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

type FileProcessor struct {
	root string
	cfg  Config
}

func NewFileProcessor(root string, cfg Config) *FileProcessor {
	return &FileProcessor{
		root: root,
		cfg:  cfg,
	}
}

func (fp *FileProcessor) ProcessFiles(out io.Writer) error {
	if len(fp.cfg.Del) > 0 {
		return fp.deleteFiles(out)
	}

	return fp.listFiles(out)
}

func (fp *FileProcessor) listFiles(out io.Writer) error {
	fileSystem := os.DirFS(fp.root)

	return fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if fp.shouldKeep(path, info) {
			return listFile(path, out)
		}

		return nil
	})
}

func (fp *FileProcessor) deleteFiles(out io.Writer) error {
	if len(fp.cfg.Del) == 0 {
		return ErrNoFilesToDelete
	}

	var filesToDelete []string
	for _, file := range fp.cfg.Del {
		full := filepath.Join(fp.root, file)
		filesToDelete = append(filesToDelete, full)
	}

	msg, err := deleteFiles(filesToDelete)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, msg)
	return nil
}

func (fp *FileProcessor) shouldKeep(path string, info fs.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	if info.Size() < fp.cfg.Size {
		return false
	}

	if len(fp.cfg.Ext) == 0 {
		return true
	}

	fileExt := filepath.Ext(path)
	return slices.Contains(fp.cfg.Ext, fileExt)
}

func listFile(path string, out io.Writer) error {
	_, err := fmt.Fprintln(out, path)
	return err
}

func deleteFiles(files []string) (string, error) {
	var deleted []string
	var errs []error

	for _, file := range files {
		if err := os.Remove(file); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file, err))
			continue
		}
		deleted = append(deleted, file)
	}

	msg := fmt.Sprintf("successfully deleted: %v", deleted)

	if len(errs) > 0 {
		return msg, fmt.Errorf("some files failed: %v", errs)
	}

	return msg, nil
}
