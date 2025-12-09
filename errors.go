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
