//go:build js

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

package jsutil

import (
	"fmt"
	"syscall/js"
	"time"
)

// console is the default 'console' object for the browser.
var console = js.Global().Get("console")

// Log logs general information to the Javascript Console.
func Log(format string, objs ...interface{}) {
	console.Call("log", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}

// LogError logs an error to the Javascript Console.
func LogError(format string, objs ...interface{}) {
	console.Call("error", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}

// LogDebug logs a debug message to the Javascript Console.
func LogDebug(format string, objs ...interface{}) {
	console.Call("debug", time.Now().Format(time.StampMilli), fmt.Sprintf(format, objs...))
}
