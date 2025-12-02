package main

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileProcessor_ShouldKeep(t *testing.T) {
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
			cfg := Config{Ext: tt.exts, Size: tt.minSize}
			processor := NewFileProcessor(".", cfg)
			result := processor.shouldKeep(tt.path, tt.info)
			if result != tt.expected {
				t.Errorf("shouldKeep(%q, %v, %d) = %v, want %v",
					tt.path, tt.exts, tt.minSize, result, tt.expected)
			}
		})
	}
}

func TestFileProcessor_ListFiles(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantOutput string
	}{
		{
			name:       "list all files no filters",
			cfg:        Config{Ext: []string{}, Size: 0},
			wantOutput: "file.go\nfile.txt\nsubdir/nested/config.yml\nsubdir/script.py\n",
		},
		{
			name:       "filter by extension .go",
			cfg:        Config{Ext: []string{".go"}, Size: 0},
			wantOutput: "file.go\n",
		},
		{
			name:       "filter by multiple extensions",
			cfg:        Config{Ext: []string{".go", ".txt"}, Size: 0},
			wantOutput: "file.go\nfile.txt\n",
		},
		{
			name:       "filter by size",
			cfg:        Config{Ext: []string{}, Size: 10},
			wantOutput: "file.go\nsubdir/nested/config.yml\nsubdir/script.py\n",
		},
		{
			name:       "no files match criteria",
			cfg:        Config{Ext: []string{".java"}, Size: 0},
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor := NewFileProcessor("./testdata", tt.cfg)
			var out bytes.Buffer
			err := processor.listFiles(&out)

			if err != nil {
				t.Errorf("listFiles() unexpected error: %v", err)
				return
			}

			got := normalizeLines(out.String())
			want := normalizeLines(tt.wantOutput)

			if got != want {
				t.Errorf("listFiles() output = %q, want %q", got, want)
			}
		})
	}
}

func TestFileProcessor_DeleteFiles(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     []string
		inputFiles     []string
		wantErr        bool
		wantDeletedNum int
	}{
		{
			name:           "all files deleted successfully",
			setupFiles:     []string{"a.txt", "b.txt"},
			inputFiles:     []string{}, // will be set to setupFiles
			wantErr:        false,
			wantDeletedNum: 2,
		},
		{
			name:           "file does not exist",
			setupFiles:     []string{},
			inputFiles:     []string{"missing.txt"},
			wantErr:        true,
			wantDeletedNum: 0,
		},
		{
			name:           "some files deleted, some fail",
			setupFiles:     []string{"a.txt"},
			inputFiles:     []string{"a.txt", "missing.txt"},
			wantErr:        true,
			wantDeletedNum: 1,
		},
		{
			name:           "empty list",
			setupFiles:     []string{},
			inputFiles:     []string{},
			wantErr:        false,
			wantDeletedNum: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			var existing []string
			for _, name := range tc.setupFiles {
				path := filepath.Join(tempDir, name)
				err := os.WriteFile(path, []byte("test"), 0644)
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				existing = append(existing, path)
			}

			input := tc.inputFiles
			if len(input) == 0 && len(existing) > 0 {
				input = existing
			} else {
				// Convert relative paths to absolute for non-existing files
				var absInput []string
				for _, f := range input {
					if !filepath.IsAbs(f) {
						absInput = append(absInput, filepath.Join(tempDir, f))
					} else {
						absInput = append(absInput, f)
					}
				}
				input = absInput
			}

			_, err := deleteFiles(input)

			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			deletedCount := 0
			for _, f := range input {
				// Check if file existed before (in setupFiles) and now doesn't exist
				wasInSetup := false
				for _, setup := range tc.setupFiles {
					if filepath.Base(f) == setup {
						wasInSetup = true
						break
					}
				}

				if wasInSetup {
					if _, statErr := os.Stat(f); statErr != nil {
						deletedCount++
					}
				}
			}

			if deletedCount != tc.wantDeletedNum {
				t.Errorf("expected %d deleted, got %d", tc.wantDeletedNum, deletedCount)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name string
		list bool
		ext  string
		del  string
		size int64
		want Config
	}{
		{
			name: "basic config",
			list: true,
			ext:  ".go .txt",
			del:  "file1 file2",
			size: 100,
			want: Config{
				List: true,
				Ext:  []string{".go", ".txt"},
				Del:  []string{"file1", "file2"},
				Size: 100,
			},
		},
		{
			name: "empty extensions and deletions",
			list: false,
			ext:  "",
			del:  "",
			size: 0,
			want: Config{
				List: false,
				Ext:  nil,
				Del:  nil,
				Size: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseConfig(tt.list, tt.ext, tt.del, tt.size)
			if !equalConfigs(got, tt.want) {
				t.Errorf("ParseConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{Size: 0},
			wantErr: false,
		},
		{
			name:    "invalid negative size",
			cfg:     Config{Size: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRoot(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		wantErr bool
	}{
		{
			name:    "empty root",
			root:    "",
			wantErr: true,
		},
		{
			name:    "non-existent root",
			root:    "/non/existent/path",
			wantErr: true,
		},
		{
			name:    "file instead of directory",
			root:    "./main.go",
			wantErr: true,
		},
		{
			name:    "valid directory",
			root:    "./testdata",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRoot(tt.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRoot() error = %v, wantErr %v", err, tt.wantErr)
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
func (m mockFileInfo) Sys() any           { return nil }

func normalizeLines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func equalConfigs(a, b Config) bool {
	if a.List != b.List || a.Size != b.Size {
		return false
	}
	if len(a.Ext) != len(b.Ext) {
		return false
	}
	if len(a.Del) != len(b.Del) {
		return false
	}
	for i := range a.Ext {
		if a.Ext[i] != b.Ext[i] {
			return false
		}
	}
	for i := range a.Del {
		if a.Del[i] != b.Del[i] {
			return false
		}
	}
	return true
}
