package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log"
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
	delLogger := log.New(fp.cfg.wLog, "DELETED FILE: ", log.LstdFlags)

	if len(fp.cfg.Del) > 0 {
		return fp.deleteFiles(out, delLogger)
	}

	if fp.cfg.archive != "" {
		return fp.archiveFiles(out)
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

func (fp *FileProcessor) deleteFiles(out io.Writer, delLogger *log.Logger) error {
	if len(fp.cfg.Del) == 0 {
		return ErrNoFilesToDelete
	}

	var filesToDelete []string
	for _, file := range fp.cfg.Del {
		full := filepath.Join(fp.root, file)
		filesToDelete = append(filesToDelete, full)
	}

	msg, err := deleteFiles(filesToDelete, delLogger)
	if err != nil {
		return err
	}

	fmt.Fprintln(out, msg)
	return nil
}

func (fp *FileProcessor) archiveFiles(out io.Writer) error {
	if fp.cfg.archive == "" {
		return ErrNoArchiveDir
	}

	fileSystem := os.DirFS(fp.root)

	var archived []string

	err := fs.WalkDir(fileSystem, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if fp.shouldKeep(path, info) {
			if err := fp.archiveFile(filepath.Join(fp.root, path)); err != nil {
				return fmt.Errorf("failed to archive %s: %w", path, err)
			}
			archived = append(archived, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Successfully archived %d files to %s\n", len(archived), filepath.Base(fp.cfg.archive))

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

func (fp *FileProcessor) archiveFile(path string) error {
	info, err := os.Stat(fp.cfg.archive)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", fp.cfg.archive)
	}

	relDir, err := filepath.Rel(fp.root, filepath.Dir(path))
	if err != nil {
		return err
	}
	dest := fmt.Sprintf("%s.gz", filepath.Base(path))
	targetpath := filepath.Join(fp.cfg.archive, relDir, dest)

	if err := os.MkdirAll(filepath.Dir(targetpath), 0755); err != nil {
		return err
	}

	// destination file
	out, err := os.OpenFile(targetpath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	// file to be archived
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()

	// create compressor
	zw := gzip.NewWriter(out)
	zw.Name = filepath.Base(path)

	// copy data
	if _, err := io.Copy(zw, in); err != nil {
		return err
	}

	// close the compressor
	if err := zw.Close(); err != nil {
		return err
	}

	return out.Close()
}

func listFile(path string, out io.Writer) error {
	_, err := fmt.Fprintln(out, path)
	return err
}

func deleteFiles(files []string, delLogger *log.Logger) (string, error) {
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
	delLogger.Println(deleted)

	if len(errs) > 0 {
		return msg, fmt.Errorf("some files failed: %v", errs)
	}

	return msg, nil
}
