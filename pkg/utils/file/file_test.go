package file

import (
	"os"
	"path/filepath"
	"testing"
)

var content = []byte("test content")

func cleanup(dir string) error {
	return os.RemoveAll(dir)
}

func TestGetFilesInDir(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (string, error)
		expected []string
	}{
		{
			name: "empty directory",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "test-empty-dir")
				return dir, err
			},
			expected: []string{},
		},
		{
			name: "directory with files",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "test-dir-with-files")
				if err != nil {
					return "", err
				}

				files := []string{"file1.txt", "file2.txt"}
				for _, file := range files {
					err = os.WriteFile(filepath.Join(dir, file), content, 0644)
					if err != nil {
						return "", err
					}
				}
				return dir, nil
			},
			expected: []string{"file1.txt", "file2.txt"},
		},
		{
			name: "nested directories with files",
			setup: func() (string, error) {
				dir, err := os.MkdirTemp("", "test-nested-dir")
				if err != nil {
					return "", err
				}

				subDir := filepath.Join(dir, "subdir")
				err = os.Mkdir(subDir, 0755)
				if err != nil {
					return "", err
				}
				err = os.WriteFile(filepath.Join(dir, "file1.txt"), content, 0644)
				if err != nil {
					return "", err
				}
				err = os.WriteFile(filepath.Join(subDir, "file2.txt"), content, 0644)
				if err != nil {
					return "", err
				}
				err = os.WriteFile(filepath.Join(subDir, "file3.txt"), content, 0644)
				if err != nil {
					return "", err
				}
				return dir, nil
			},
			expected: []string{"file1.txt", "subdir/file2.txt", "subdir/file3.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test directory and files
			dir, err := tt.setup()
			if err != nil {
				t.Fatalf("Setup failed: %v", err)
			}
			defer cleanup(dir)

			files, err := GetFilesInDir(dir)
			if err != nil {
				t.Fatalf("GetFilesInDir() error = %v", err)
			}

			// Validate the result
			for i, file := range files {
				if file != filepath.Join(dir, tt.expected[i]) {
					t.Errorf("Expected file %v, got %v", tt.expected[i], file)
				}
			}
		})
	}
}
