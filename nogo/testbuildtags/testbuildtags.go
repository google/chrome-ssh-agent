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

// Package testbuildtags ensures that go files with tests defined in them do not
// have any build tags configured.
//
// Build tags, if not extremely carefully used, may prevent those files from
// being included in the test package and thus silently prevent them from
// being executed.
//
//nolint:godox
// TODO: Find a way to re-enable this static analyzer. See
//	https://github.com/google/chrome-ssh-agent/issues/163
package testbuildtags

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "testbuildtags",
	Doc:  "reports build tags present in files that contain tests",
	Run:  run,
}

// reportBuildTag reports occurrence of any build tag that is found. Build tags
// have particular rules about where they can be located within a file, but for
// simplicity we just complain about things that might *look* like build tags
// even if Go would not treat them as such.  This is a small enough code base
// that we can adjust if this presents a problem.
//
// See https://pkg.go.dev/cmd/go#hdr-Build_constraints for build tag format.
func reportBuildTag(pass *analysis.Pass, f *ast.File) {
	for _, group := range f.Comments {
		for _, line := range group.List {
			if strings.Contains(line.Text, "//go:build") {
				pass.Reportf(line.Pos(), "build tags not permitted inside tests")
			}
		}
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		// We only care about tests.
		if !strings.HasSuffix(pass.Fset.File(f.Pos()).Name(), "_test.go") {
			continue
		}
		reportBuildTag(pass, f)
	}
	return nil, nil
}
