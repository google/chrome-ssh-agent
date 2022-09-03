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

package dom

import (
	"syscall/js"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func init() {
	// Make URLSearchParams JavaScript implementation available in unit
	// tests, which run under node.js.  It is available by default when
	// running inside of browsers.
	js.Global().Call("eval", `
		var URLSearchParams = require('@ungap/url-search-params');
	`)
}

func TestHas(t *testing.T) {
	testcases := []struct {
		description string
		queryString string
		param       string
		want        bool
	}{
		{
			description: "param with value",
			queryString: "?key=value",
			param:       "key",
			want:        true,
		},
		{
			description: "param without value",
			queryString: "?key",
			param:       "key",
			want:        true,
		},
		{
			description: "no param found",
			queryString: "?other-key",
			param:       "key",
			want:        false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			qs := NewURLSearchParams(tc.queryString)
			if diff := cmp.Diff(qs.Has(tc.param), tc.want); diff != "" {
				t.Errorf("incorrect result; -got +want: %s", diff)
			}
		})
	}
}
