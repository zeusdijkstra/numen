package main

import (
	"os"
	"strings"
)

type Config struct {
	List bool
	Ext  []string
	Size int64
	Del  []string
}

func ParseConfig(list bool, ext, del string, size int64) Config {
	return Config{
		List: list,
		Ext:  parseExtensions(ext),
		Size: size,
		Del:  parseExtensions(del),
	}
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
