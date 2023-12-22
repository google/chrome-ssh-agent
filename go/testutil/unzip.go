// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CleanupFunc can be invoked to cleanup any temporary state.
type CleanupFunc func()

const unzipChunkSizeBytes = 4096

// See https://github.com/securego/gosec/issues/324#issuecomment-935927967
func sanitizeArchivePath(dir, fileName string) (string, error) {
	fullPath := filepath.Join(dir, fileName)
	if !strings.Contains(fullPath, filepath.Clean(dir)) {
		return "", fmt.Errorf("Archive contained unsafe path: %s", fileName)
	}

	return fullPath, nil
}

// UnzipTemp unzips a zip archive to temporary path, and returns the path.
// The returned cleanup function should be invoked to cleanup any temporary
// state when it is no longer needed.
func UnzipTemp(path string) (string, CleanupFunc, error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", nil, fmt.Errorf("Failed to create temp directory: %w", err)
	}

	rdr, err := zip.OpenReader(path)
	if err != nil {
		defer os.RemoveAll(dir)
		return "", nil, fmt.Errorf("Failed to open extension file: %w", err)
	}
	defer rdr.Close()

	for _, f := range rdr.File {
		err := func() error {
			filePath, err := sanitizeArchivePath(dir, f.Name)
			if err != nil {
				return fmt.Errorf("Archive contained unsafe path: %w", err)
			}

			if f.Mode().IsDir() {
				if merr := os.MkdirAll(filePath, os.ModePerm); merr != nil {
					return fmt.Errorf("Failed to create destination directory %s: %w", filepath.Dir(filePath), merr)
				}
				return nil
			}

			if merr := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); merr != nil {
				return fmt.Errorf("Failed to create destination directory %s: %w", filepath.Dir(filePath), merr)
			}

			dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("Failed to open destination file %s: %w", filePath, err)
			}
			defer dstFile.Close()

			archFile, err := f.Open()
			if err != nil {
				return fmt.Errorf("Failed to open source file %s: %w", f.Name, err)
			}
			defer archFile.Close()

			for {
				if _, err := io.CopyN(dstFile, archFile, unzipChunkSizeBytes); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return fmt.Errorf("Failed to copy to destination file %s: %w", filePath, err)
				}
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
