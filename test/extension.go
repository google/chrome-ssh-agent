package test

import (
	"fmt"
	"io"
	"os"

	"archive/zip"
	"net/url"
	"path/filepath"
)

const (
	extensionId = "eechpbnaifiimgajnomdipfaamobdfha"
)

func makeExtensionUrl(relPath string, queryString string) *url.URL {
	return &url.URL{
		Scheme:   "chrome-extension",
		Host:     extensionId,
		Path:     relPath,
		RawQuery: queryString,
	}
}

type cleanupFunc func()

func unzipExtension(path string) (string, cleanupFunc, error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", nil, fmt.Errorf("Failed to create temp directory: %v", err)
	}

	rdr, err := zip.OpenReader(path)
	if err != nil {
		defer os.RemoveAll(dir)
		return "", nil, fmt.Errorf("Failed to open extension file: %v", err)
	}
	defer rdr.Close()

	for _, f := range rdr.File {
		filePath := filepath.Join(dir, f.Name)

		err := func() error {
			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				return fmt.Errorf("Failed to create destination directory %s: %v", filepath.Dir(filePath), err)
			}

			dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("Failed to open destination file %s: %v", filePath, err)
			}
			defer dstFile.Close()

			archFile, err := f.Open()
			if err != nil {
				return fmt.Errorf("Failed to open source file %s: %v", f.Name, err)
			}
			defer archFile.Close()

			if _, err := io.Copy(dstFile, archFile); err != nil {
				return fmt.Errorf("Failed to copy to destination file %s: %v", filePath, err)
			}
			return nil
		}()
		if err != nil {
			defer os.RemoveAll(dir)
			return "", nil, err
		}
	}

	return dir, func() { os.RemoveAll(dir) }, nil
}
