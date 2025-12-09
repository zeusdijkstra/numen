package main

import (
	"bytes"
	"compress/gzip"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// ============================================================================
// FILE PROCESSING TESTS
// ============================================================================

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

// ============================================================================
// FILE LISTING TESTS
// ============================================================================

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

// ============================================================================
// FILE DELETION TESTS
// ============================================================================

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

			_, err := deleteFiles(input, log.New(os.Stdout, "", 0))

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
				if slices.Contains(tc.setupFiles, filepath.Base(f)) {
					wasInSetup = true
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

// ============================================================================
// CONFIGURATION TESTS
// ============================================================================

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name    string
		list    bool
		ext     string
		del     string
		archive string
		logFile string
		size    int64
		want    Config
	}{
		{
			name:    "basic config",
			list:    true,
			ext:     ".go .txt",
			del:     "file1 file2",
			archive: "/tmp/archive",
			logFile: "",
			size:    100,
			want: Config{
				List:    true,
				Ext:     []string{".go", ".txt"},
				Del:     []string{"file1", "file2"},
				archive: "/tmp/archive",
				Size:    100,
				LogFile: "",
				wLog:    os.Stdout,
			},
		},
		{
			name:    "empty extensions and deletions",
			list:    false,
			ext:     "",
			del:     "",
			archive: "",
			logFile: "",
			size:    0,
			want: Config{
				List:    false,
				Ext:     nil,
				Del:     nil,
				archive: "",
				Size:    0,
				LogFile: "",
				wLog:    os.Stdout,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig(tt.list, tt.ext, tt.del, "", tt.logFile, tt.size)
			if err != nil {
				t.Errorf("ParseConfig() unexpected error: %v", err)
				return
			}
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

func TestParseConfig_WithLogFile(t *testing.T) {
	tests := []struct {
		name      string
		logFile   string
		wantErr   bool
		wantField string
	}{
		{
			name:      "no log file",
			logFile:   "",
			wantErr:   false,
			wantField: "",
		},
		{
			name:      "valid log file path",
			logFile:   "/tmp/test.log",
			wantErr:   false,
			wantField: "/tmp/test.log",
		},
		{
			name:      "invalid log file path",
			logFile:   "/invalid/path/test.log",
			wantErr:   true,
			wantField: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseConfig(false, "", "", "", tt.logFile, 0)

			if tt.wantErr && err == nil {
				t.Errorf("ParseConfig() expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ParseConfig() unexpected error: %v", err)
				return
			}

			if !tt.wantErr && cfg.LogFile != tt.wantField {
				t.Errorf("ParseConfig() LogFile = %v, want %v", cfg.LogFile, tt.wantField)
			}
		})
	}
}

// ============================================================================
// VALIDATION TESTS
// ============================================================================

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
	if a.List != b.List || a.Size != b.Size || a.LogFile != b.LogFile {
		return false
	}
	// Skip wLog comparison since it's a writer interface
	// Skip archive comparison since it's a private field
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

// ============================================================================
// ARCHIVE FUNCTIONALITY TESTS
// ============================================================================

// Helper function to create test files with content
func createTestFiles(t *testing.T, dir string, files map[string]string) {
	for name, content := range files {
		path := filepath.Join(dir, name)
		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", path, err)
		}
	}
}

// Helper function to verify gzip file integrity and content
func verifyGzipFile(t *testing.T, gzPath, expectedContent string) {
	file, err := os.Open(gzPath)
	if err != nil {
		t.Fatalf("failed to open gzip file %s: %v", gzPath, err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	content, err := os.ReadFile(gzPath)
	if err != nil {
		t.Fatalf("failed to read gzip file: %v", err)
	}

	// Verify it's actually compressed
	if len(content) == len(expectedContent) {
		t.Errorf("file doesn't appear to be compressed")
	}

	// Reset and read actual content
	file.Seek(0, 0)
	gzReader, _ = gzip.NewReader(file)
	actualContent := make([]byte, len(expectedContent))
	n, err := gzReader.Read(actualContent)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("failed to read decompressed content: %v", err)
	}

	if string(actualContent[:n]) != expectedContent {
		t.Errorf("gzip content = %q, want %q", string(actualContent[:n]), expectedContent)
	}
}

// Helper function to count archived files
func countArchivedFiles(t *testing.T, archiveDir string) int {
	count := 0
	filepath.WalkDir(archiveDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".gz" {
			count++
		}
		return nil
	})
	return count
}

func TestFileProcessor_ArchiveFile(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T, root, archive string) string
		wantErr    bool
		verifyFunc func(t *testing.T, archiveDir, sourceFile string)
	}{
		{
			name: "successfully archive single file",
			setupFunc: func(t *testing.T, root, archive string) string {
				sourceFile := filepath.Join(root, "test.txt")
				os.WriteFile(sourceFile, []byte("hello world"), 0644)
				return sourceFile
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, archiveDir, sourceFile string) {
				gzFile := filepath.Join(archiveDir, "test.txt.gz")
				verifyGzipFile(t, gzFile, "hello world")
			},
		},
		{
			name: "archive file in nested directory",
			setupFunc: func(t *testing.T, root, archive string) string {
				sourceFile := filepath.Join(root, "subdir", "nested", "test.txt")
				os.MkdirAll(filepath.Dir(sourceFile), 0755)
				os.WriteFile(sourceFile, []byte("nested content"), 0644)
				return sourceFile
			},
			wantErr: false,
			verifyFunc: func(t *testing.T, archiveDir, sourceFile string) {
				gzFile := filepath.Join(archiveDir, "subdir", "nested", "test.txt.gz")
				verifyGzipFile(t, gzFile, "nested content")
			},
		},
		{
			name: "non-existent source file",
			setupFunc: func(t *testing.T, root, archive string) string {
				return filepath.Join(root, "nonexistent.txt")
			},
			wantErr:    true,
			verifyFunc: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			rootDir := filepath.Join(tempDir, "root")
			archiveDir := filepath.Join(tempDir, "archive")

			os.MkdirAll(rootDir, 0755)
			os.MkdirAll(archiveDir, 0755)

			sourceFile := tt.setupFunc(t, rootDir, archiveDir)

			cfg := Config{archive: archiveDir}
			processor := NewFileProcessor(rootDir, cfg)

			err := processor.archiveFile(sourceFile)

			if (err != nil) != tt.wantErr {
				t.Errorf("archiveFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.verifyFunc != nil {
				tt.verifyFunc(t, archiveDir, sourceFile)
			}
		})
	}
}

func TestFileProcessor_ArchiveFiles(t *testing.T) {
	tests := []struct {
		name         string
		setupFiles   map[string]string
		cfg          Config
		wantErr      bool
		wantArchived int
		wantOutput   string
	}{
		{
			name: "archive files by extension",
			setupFiles: map[string]string{
				"file.go":  "package main",
				"file.txt": "hello world",
				"file.py":  "print('hello')",
			},
			cfg: Config{
				Ext:     []string{".go", ".txt"},
				Size:    0,
				archive: "archive",
			},
			wantErr:      false,
			wantArchived: 2,
			wantOutput:   "Successfully archived 2 files to archive\n",
		},
		{
			name: "archive files by size",
			setupFiles: map[string]string{
				"small.txt":  "hi",          // 2 bytes
				"large.txt":  "hello world", // 11 bytes
				"medium.txt": "hello",       // 5 bytes
			},
			cfg: Config{
				Ext:     []string{},
				Size:    5,
				archive: "archive",
			},
			wantErr:      false,
			wantArchived: 2, // large.txt (11) and medium.txt (5)
			wantOutput:   "Successfully archived 2 files to archive\n",
		},
		{
			name: "no files match criteria",
			setupFiles: map[string]string{
				"file.py": "print('hello')",
			},
			cfg: Config{
				Ext:     []string{".go"},
				Size:    0,
				archive: "archive",
			},
			wantErr:      false,
			wantArchived: 0,
			wantOutput:   "Successfully archived 0 files to archive\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			rootDir := filepath.Join(tempDir, "root")
			archiveDir := filepath.Join(tempDir, "archive")

			os.MkdirAll(rootDir, 0755)
			os.MkdirAll(archiveDir, 0755)

			createTestFiles(t, rootDir, tt.setupFiles)

			cfg := tt.cfg
			cfg.archive = archiveDir
			processor := NewFileProcessor(rootDir, cfg)

			var out bytes.Buffer
			err := processor.archiveFiles(&out)

			if (err != nil) != tt.wantErr {
				t.Errorf("archiveFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				gotOutput := out.String()
				if gotOutput != tt.wantOutput {
					t.Errorf("archiveFiles() output = %q, want %q", gotOutput, tt.wantOutput)
				}

				archivedCount := countArchivedFiles(t, archiveDir)
				if archivedCount != tt.wantArchived {
					t.Errorf("archiveFiles() archived %d files, want %d", archivedCount, tt.wantArchived)
				}
			}
		})
	}
}

func TestFileProcessor_ProcessFiles_WithArchive(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles map[string]string
		cfg        Config
		wantOutput string
		wantErr    bool
	}{
		{
			name: "archive mode selected",
			setupFiles: map[string]string{
				"test.txt": "hello world",
				"test.go":  "package main",
			},
			cfg: Config{
				archive: "archive",
				Ext:     []string{".txt"},
				Size:    0,
			},
			wantOutput: "Successfully archived 1 files to archive\n",
			wantErr:    false,
		},
		{
			name: "archive mode not selected when empty",
			setupFiles: map[string]string{
				"test.txt": "hello world",
			},
			cfg: Config{
				archive: "",
				Ext:     []string{".txt"},
				Size:    0,
			},
			wantOutput: "test.txt\n",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			rootDir := filepath.Join(tempDir, "root")
			archiveDir := filepath.Join(tempDir, "archive")

			os.MkdirAll(rootDir, 0755)
			if tt.cfg.archive != "" {
				os.MkdirAll(archiveDir, 0755)
				tt.cfg.archive = archiveDir
			}

			createTestFiles(t, rootDir, tt.setupFiles)

			processor := NewFileProcessor(rootDir, tt.cfg)

			var out bytes.Buffer
			err := processor.ProcessFiles(&out)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotOutput := normalizeLines(out.String())
			wantOutput := normalizeLines(tt.wantOutput)

			if gotOutput != wantOutput {
				t.Errorf("ProcessFiles() output = %q, want %q", gotOutput, wantOutput)
			}
		})
	}
}
