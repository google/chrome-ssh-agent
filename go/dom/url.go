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
	"github.com/gopherjs/gopherjs/js"
)

// URLSearchParams is a thin wrapper around the URLSearchParams API.
// See https://url.spec.whatwg.org/#urlsearchparams.
type URLSearchParams struct {
	o *js.Object
}

// DefaultQueryString returns the query string used to request the current
// document.  This is likely not available during unit tests, but is
// available in normal operation.
func DefaultQueryString() string {
	return js.Global.Get("window").Get("location").Get("search").String()
}

// NewURLSearchParams returns a URLSearchParams for the specified query string.
func NewURLSearchParams(queryString string) *URLSearchParams {
	return &URLSearchParams{
		o: js.Global.Get("URLSearchParams").New(queryString),
	}
}

// Has indicates if the query string contains the specified parameter.
func (u *URLSearchParams) Has(param string) bool {
	return u.o.Call("has", param).Bool()
}
