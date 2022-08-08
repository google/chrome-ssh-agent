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

package tools

import (
	"fmt"
	"io"
	"os"
	"archive/zip"
	"path/filepath"
)

// CleanupFunc can be invoked to cleanup any temporary state.
type CleanupFunc func()

// UnzipTemp unzips a zip archive to temporary path, and returns the path.
// The returned cleanup function should be invoked to cleanup any temporary
// state when it is no longer needed.
func UnzipTemp(path string) (string, CleanupFunc, error) {
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
