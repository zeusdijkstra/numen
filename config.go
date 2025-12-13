package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	List    bool
	Ext     []string
	Size    int64
	Del     []string
	LogFile string
	wLog    io.Writer
	archive string
	logFile *os.File
}

func (c *Config) Close() error {
	if c.logFile != nil {
		return c.logFile.Close()
	}
	return nil
}

func ParseConfig(list bool, ext, del, archive, logFile string, size int64) (*Config, error) {
	cfg := &Config{
		List:    list,
		Ext:     parseExtensions(ext),
		Size:    size,
		Del:     parseExtensions(del),
		LogFile: logFile,
		archive: archive,
	}

	if err := cfg.setupLogging(); err != nil {
		return nil, err
	}

	if err := ValidateConfig(cfg); err != nil {
		cfg.Close() // Clean up on error
		return nil, err
	}

	return cfg, nil
}

func (c *Config) setupLogging() error {
	if c.LogFile == "" {
		c.wLog = os.Stdout
		return nil
	}

	logDir := filepath.Dir(c.LogFile)
	if err := validateDirectory(logDir, ValidateLogDir); err != nil {
		return err
	}

	file, err := openFileWithFlags(c.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return NewLogFileOpenError(c.LogFile, err)
	}

	c.logFile = file
	c.wLog = io.MultiWriter(os.Stdout, file)
	return nil
}

// DirectoryValidationType defines the type of directory validation to perform
type DirectoryValidationType int

const (
	ValidateLogDir DirectoryValidationType = iota
	ValidateArchiveDir
	ValidateRootDir
)

func validateDirectory(path string, dirType DirectoryValidationType) error {
	if path == "" || path == "." {
		return nil
	}

	switch dirType {
	case ValidateLogDir:
		if err := os.MkdirAll(path, 0755); err != nil {
			return NewLogDirCreateError(path, err)
		}
		return nil

	case ValidateArchiveDir:
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return NewArchiveDirNotFoundError(path, err)
			}
			return NewArchiveDirAccessError(path, err)
		}
		if !info.IsDir() {
			return NewArchivePathNotDirError(path)
		}
		return nil

	case ValidateRootDir:
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ErrRootNotFound
			}
			if os.IsPermission(err) {
				return NewRootPermissionError(path, err)
			}
			return NewRootAccessError(path, err)
		}
		if !info.IsDir() {
			return ErrRootNotDir
		}

		testPath := filepath.Join(path, ".test_access")
		if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
			if os.IsPermission(err) {
				return NewRootNoWritePermissionError(path, err)
			}
		} else {
			os.Remove(testPath)
		}
		return nil

	default:
		return fmt.Errorf("unknown directory validation type: %d", dirType)
	}
}

func parseExtensions(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	exts := strings.Fields(input)
	if len(exts) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(exts))
	seen := make(map[string]bool)

	for _, ext := range exts {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}

		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		ext = strings.ToLower(ext)

		if !seen[ext] {
			normalized = append(normalized, ext)
			seen[ext] = true
		}
	}

	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func ValidateConfig(cfg *Config) error {
	if cfg.Size < 0 {
		return ErrInvalidSize
	}

	if cfg.archive != "" {
		archiveDir := filepath.Dir(cfg.archive)
		if err := validateDirectory(archiveDir, ValidateArchiveDir); err != nil {
			return err
		}
	}

	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, ext := range append(cfg.Ext, cfg.Del...) {
		for _, char := range invalidChars {
			if strings.Contains(ext, char) {
				return NewInvalidExtensionError(ext, char)
			}
		}
	}

	return nil
}

func ValidateRoot(root string) error {
	root = strings.TrimSpace(root)

	if root == "" {
		return ErrEmptyRoot
	}

	root = filepath.Clean(root)

	if err := validateDirectory(root, ValidateRootDir); err != nil {
		return err
	}

	return nil
}
