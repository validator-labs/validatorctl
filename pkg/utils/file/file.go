// Package file provides utility functions for working with a filesystem.
package file

import (
	"io/fs"
	"path/filepath"
)

// GetFilesInDir walks the file tree from dir and returns a list of all files found in lexical order.
func GetFilesInDir(dir string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}
