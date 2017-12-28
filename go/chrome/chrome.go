// Copyright 2017 Google LLC
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

package chrome

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

var (
	C           = js.Global.Get("chrome")
	Runtime     = C.Get("runtime")
	Storage     = C.Get("storage")
	SyncStorage = Storage.Get("sync")
	ExtensionId = Runtime.Get("id").String()
)

func LastError() error {
	if err := Runtime.Get("lastError"); err != nil && err != js.Undefined {
		return errors.New(err.Get("message").String())
	}
	return nil
}
