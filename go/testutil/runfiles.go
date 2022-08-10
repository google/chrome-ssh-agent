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
	"fmt"
	"os"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
)

// MustRunfile returns the path to the specified runfile. Panic on error.
func MustRunfile(path string) string {
	path, err := bazel.Runfile(path)
	if err != nil {
		panic(fmt.Errorf("failed to find runfile %s: %v", path, err))
	}
	return path
}

// MustReadRunfile returns the contents of the specified runfile. Panic on
// error.
func MustReadRunfile(path string) []byte {
	fullPath := MustRunfile(path)
	buf, err := os.ReadFile(fullPath)
	if err != nil {
		panic(fmt.Errorf("failed to read runfile %s: %v", fullPath, err))
	}
	return buf
}
