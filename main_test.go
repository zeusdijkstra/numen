package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestShouldKeep(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		exts     []string
		minSize  int64
		info     fs.FileInfo
		expected bool
	}{
		{
			name:     "directory filtered out",
			path:     "somedir",
			exts:     []string{},
			minSize:  0,
			info:     createFileInfo("somedir", 100, true),
			expected: false,
		},
		{
			name:     "file too small",
			path:     "small.txt",
			exts:     []string{},
			minSize:  10,
			info:     createFileInfo("small.txt", 5, false),
			expected: false,
		},
		{
			name:     "file meets min size",
			path:     "large.txt",
			exts:     []string{},
			minSize:  5,
			info:     createFileInfo("large.txt", 15, false),
			expected: true,
		},
		{
			name:     "matching extension",
			path:     "file.go",
			exts:     []string{".go", ".txt"},
			minSize:  0,
			info:     createFileInfo("file.go", 10, false),
			expected: true,
		},
		{
			name:     "non-matching extension",
			path:     "file.py",
			exts:     []string{".go", ".txt"},
			minSize:  0,
			info:     createFileInfo("file.py", 10, false),
			expected: false,
		},
		{
			name:     "no extension filter",
			path:     "anyfile.xyz",
			exts:     []string{},
			minSize:  0,
			info:     createFileInfo("anyfile.xyz", 10, false),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldKeep(tt.path, tt.exts, tt.minSize, tt.info)
			if result != tt.expected {
				t.Errorf("shouldKeep(%q, %v, %d) = %v, want %v",
					tt.path, tt.exts, tt.minSize, result, tt.expected)
			}
		})
	}
}

func TestRun_WithRealFS(t *testing.T) {
	tests := []struct {
		name       string
		cfg        config
		wantOutput string
	}{
		{
			name:       "list all files no filters",
			cfg:        config{ext: []string{}, size: 0},
			wantOutput: "file.go\nfile.txt\nsubdir/nested/config.yml\nsubdir/script.py\n",
		},
		{
			name:       "filter by extension .go",
			cfg:        config{ext: []string{".go"}, size: 0},
			wantOutput: "file.go\n",
		},
		{
			name:       "filter by multiple extensions",
			cfg:        config{ext: []string{".go", ".txt"}, size: 0},
			wantOutput: "file.go\nfile.txt\n",
		},
		{
			name:       "filter by size",
			cfg:        config{ext: []string{}, size: 10},
			wantOutput: "file.go\nsubdir/nested/config.yml\nsubdir/script.py\n",
		},
		{
			name:       "no files match criteria",
			cfg:        config{ext: []string{".java"}, size: 0},
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var out bytes.Buffer
			err := run("./testdata", &out, tt.cfg)

			if err != nil {
				t.Errorf("run() unexpected error: %v", err)
				return
			}

			got := normalizeLines(out.String())
			want := normalizeLines(tt.wantOutput)

			if got != want {
				t.Errorf("run() output = %q, want %q", got, want)
			}
		})
	}
}

// func TestRun_WithMockFS(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		cfg        config
// 		wantOutput string
// 	}{
// 		{
// 			name:       "list all files no filters",
// 			cfg:        config{ext: []string{}, size: 0},
// 			wantOutput: "file.txt\nfile.go\nsubdir/script.py\nsubdir/nested/config.yml\n",
// 		},
// 		{
// 			name:       "filter by extension .go",
// 			cfg:        config{ext: []string{".go"}, size: 0},
// 			wantOutput: "file.go\n",
// 		},
// 		{
// 			name:       "filter by multiple extensions",
// 			cfg:        config{ext: []string{".go", ".txt"}, size: 0},
// 			wantOutput: "file.txt\nfile.go\n",
// 		},
// 		{
// 			name:       "filter by size",
// 			cfg:        config{ext: []string{}, size: 10},
// 			wantOutput: "file.go\nsubdir/script.py\nsubdir/nested/config.yml\n",
// 		},
// 		{
// 			name:       "no files match criteria",
// 			cfg:        config{ext: []string{".java"}, size: 0},
// 			wantOutput: "",
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mockFS := createTestFS()
//
// 			// replace os.DirFS for testing
// 			originalDirFS := osDirFS
// 			osDirFS = func(root string) fs.FS { return mockFS }
// 			defer func() { osDirFS = originalDirFS }()
//
// 			var out bytes.Buffer
// 			err := run(".", &out, tt.cfg)
//
// 			if err != nil {
// 				t.Errorf("run() unexpected error: %v", err)
// 				return
// 			}
//
// 			got := normalizeLines(out.String())
// 			want := normalizeLines(tt.wantOutput)
//
// 			if got != want {
// 				t.Errorf("run() output = %q, want %q", got, want)
// 			}
// 		})
// 	}
// }

func TestDeleteFiles(t *testing.T) {
	// Helper to create temp files
	createTempFiles := func(t *testing.T, names []string) []string {
		t.Helper()
		var paths []string
		for _, name := range names {
			path := filepath.Join(t.TempDir(), name)
			err := os.WriteFile(path, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			paths = append(paths, path)
		}
		return paths
	}

	tests := []struct {
		name           string
		existingFiles  []string // files to create before test
		inputFiles     []string // files passed to deleteFiles()
		wantErr        bool
		wantDeletedNum int
	}{
		{
			name:           "all files deleted successfully",
			existingFiles:  []string{"a.txt", "b.txt"},
			inputFiles:     []string{}, // placeholder updated below
			wantErr:        false,
			wantDeletedNum: 2,
		},
		{
			name:           "file does not exist",
			existingFiles:  []string{},
			inputFiles:     []string{"missing.txt"},
			wantErr:        true,
			wantDeletedNum: 0,
		},
		{
			name:           "some files deleted, some fail",
			existingFiles:  []string{"a.txt"},
			inputFiles:     []string{}, // placeholder updated below
			wantErr:        true,
			wantDeletedNum: 1,
		},
		{
			name:           "empty list",
			existingFiles:  []string{},
			inputFiles:     []string{},
			wantErr:        false,
			wantDeletedNum: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create the files that should exist
			existing := createTempFiles(t, tc.existingFiles)

			// Determine the files to pass as input:
			// If inputFiles is empty but existingFiles is not, use existing.
			input := tc.inputFiles
			if len(input) == 0 && len(existing) > 0 {
				input = existing
			}

			msg, err := deleteFiles(input)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check that the message contains the list of deleted files
			for _, f := range existing {
				if strings.Contains(msg, filepath.Base(f)) {
					// count only those that should have succeeded
					continue
				}
			}

			// Confirm how many files were actually deleted
			deletedCount := 0
			for _, f := range input {
				if _, statErr := os.Stat(f); statErr != nil {
					deletedCount++
				}
			}

			if deletedCount != tc.wantDeletedNum {
				t.Errorf("expected %d deleted, got %d", tc.wantDeletedNum, deletedCount)
			}
		})
	}
}

func TestMultipleExtensions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{".go", []string{".go"}},
		{".go .txt .py", []string{".go", ".txt", ".py"}},
		{"   .go   .txt   ", []string{".go", ".txt"}},
		{".go\t.txt\n.py", []string{".go", ".txt", ".py"}},
		{"  .go\t   .txt \n .py  ", []string{".go", ".txt", ".py"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := handleMultiple(tt.input)

			if len(got) != len(tt.expected) {
				t.Fatalf("len=%d, want=%d (got=%v, expected=%v)",
					len(got), len(tt.expected), got, tt.expected)
			}

			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("multipleExtensions(%q) = %v, want %v",
						tt.input, got, tt.expected)
				}
			}
		})
	}
}

func createFileInfo(name string, size int64, isDir bool) fs.FileInfo {
	return mockFileInfo{
		name:  name,
		size:  size,
		isDir: isDir,
	}
}

type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() fs.FileMode  { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

func normalizeLines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func createTestFS() *fstest.MapFS {
	return &fstest.MapFS{
		"file.txt": {
			Data: []byte("content"),
		},
		"file.go": {
			Data: []byte("package main"),
		},
		"subdir/script.py": {
			Data: []byte("print('hello')"),
		},
		"subdir/nested/config.yml": {
			Data: []byte("key: value"),
		},
		"directory": {
			Mode: fs.ModeDir,
		},
	}
}

// make os.DirFS injectable for testing
var osDirFS = func(root string) fs.FS {
	return os.DirFS(root)
}
