package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type Config struct {
	List    bool
	Ext     []string
	Size    int64
	Del     []string
	LogFile string
	wLog    io.Writer
}

func ParseConfig(list bool, ext, del string, size int64, logFile string) (Config, error) {
	var writer io.Writer = os.Stdout

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return Config{}, fmt.Errorf("failed to open log file %s: %w", logFile, err)
		}

		writer = io.MultiWriter(os.Stdout, file)
	}

	return Config{
		List:    list,
		wLog:    writer,
		Ext:     parseExtensions(ext),
		Size:    size,
		Del:     parseExtensions(del),
		LogFile: logFile,
	}, nil
}

func parseExtensions(input string) []string {
	if input == "" {
		return nil
	}

	exts := strings.Fields(input)
	if len(exts) == 0 {
		return nil
	}

	return exts
}

func ValidateConfig(cfg Config) error {
	if cfg.Size < 0 {
		return ErrInvalidSize
	}

	return nil
}

func ValidateRoot(root string) error {
	if root == "" {
		return ErrEmptyRoot
	}

	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrRootNotFound
		}
		return err
	}

	if !info.IsDir() {
		return ErrRootNotDir
	}

	return nil
}
