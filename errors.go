package main

import (
	"fmt"
)

type AppError struct {
	Code    string
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

var (
	ErrInvalidSize = &AppError{
		Code:    "INVALID_SIZE",
		Message: "file size cannot be negative",
	}
	ErrEmptyRoot = &AppError{
		Code:    "EMPTY_ROOT",
		Message: "root directory cannot be empty",
	}
	ErrRootNotFound = &AppError{
		Code:    "ROOT_NOT_FOUND",
		Message: "root directory does not exist",
	}
	ErrRootNotDir = &AppError{
		Code:    "ROOT_NOT_DIR",
		Message: "root path must be a directory",
	}
	ErrNoFilesToDelete = &AppError{
		Code:    "NO_FILES_TO_DELETE",
		Message: "no files specified for deletion",
	}
	ErrNoArchiveDir = &AppError{
		Code:    "NO_ARCHIVE_DIR",
		Message: "no archive dir specified",
	}
)

// NewPathError creates a generic path-related error
func NewPathError(code string, path string, operation string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf("%s: %s", operation, path),
		Err:     err,
	}
}

// Log file errors
func NewLogDirCreateError(dir string, err error) *AppError {
	return NewPathError("LOG_DIR_CREATE_FAILED", dir, "failed to create log directory", err)
}

func NewLogFileOpenError(file string, err error) *AppError {
	return NewPathError("LOG_FILE_OPEN_FAILED", file, "failed to open log file", err)
}

// Archive errors
func NewArchiveDirNotFoundError(dir string, err error) *AppError {
	return NewPathError("ARCHIVE_DIR_NOT_FOUND", dir, "archive directory does not exist", err)
}

func NewArchiveDirAccessError(dir string, err error) *AppError {
	return NewPathError("ARCHIVE_DIR_ACCESS_FAILED", dir, "failed to access archive directory", err)
}

func NewArchivePathNotDirError(path string) *AppError {
	return &AppError{
		Code:    "ARCHIVE_PATH_NOT_DIR",
		Message: fmt.Sprintf("archive path parent is not a directory: %s", path),
	}
}

// Extension validation errors
func NewInvalidExtensionError(ext, char string) *AppError {
	return &AppError{
		Code:    "INVALID_EXTENSION",
		Message: fmt.Sprintf("extension %s contains invalid character: %s", ext, char),
	}
}

// Root directory errors
func NewRootPermissionError(root string, err error) *AppError {
	return NewPathError("ROOT_PERMISSION_DENIED", root, "permission denied accessing root", err)
}

func NewRootAccessError(root string, err error) *AppError {
	return NewPathError("ROOT_ACCESS_FAILED", root, "failed to access root", err)
}

func NewRootNoWritePermissionError(root string, err error) *AppError {
	return NewPathError("ROOT_NO_WRITE_PERMISSION", root, "no write permission for root directory", err)
}
