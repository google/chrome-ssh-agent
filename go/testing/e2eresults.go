//go:build js

// Copyright 2018 Google LLC
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

// Package testing implements utilities to aid in testing.
package testing

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/dom"
)

// resultsAsString returns the test results as a single string suitable for
// display.
func resultsAsString(errs []error) string {
	var lines []string
	for _, err := range errs {
		lines = append(lines, err.Error())
	}
	return strings.Join(lines, "\n")
}

// getBody returns the body object, and undefined if none is present.
func getBody(d *dom.Doc) js.Value {
	for _, e := range d.GetElementsByTag("body") {
		return e
	}
	return js.Undefined()
}

// WriteResults adds elements to the supplied DOM summarizing the test results.
// The elements are given identifiers such that the results can be queried by
// automation. The following elements are added:
// - failureCount: a div element, whose contained text is the number of tests
//     that failed.
// - failures: A pre element, whose contained text is a human-readable list
//     of the individual failures.
func WriteResults(d *dom.Doc, errs []error) {
	body := getBody(d)
	// Top-level container element into which we'll write results.
	dom.AppendChild(body, d.NewElement("div"), func(results js.Value) {
		// Indicate how many tests failed.
		dom.AppendChild(results, d.NewElement("div"), func(failureCount js.Value) {
			// Allow the element to be read by automation.
			failureCount.Set("id", "failureCount")
			dom.AppendChild(failureCount, d.NewText(fmt.Sprintf("%d", len(errs))), nil)
		})

		// Enumerate the failures. This is a more readable list of the
		// individual tests that failed.
		dom.AppendChild(results, d.NewElement("pre"), func(failures js.Value) {
			// Allow element to be read by automation.
			failures.Set("id", "failures")
			dom.AppendChild(failures, d.NewText(resultsAsString(errs)), nil)
		})
	})
}
