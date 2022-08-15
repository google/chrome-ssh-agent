//go:build js && wasm

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

package message

import (
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/jsutil"
)

// Sender specifies the interface for a type that sends messages.
type Sender interface {
	// Send sends a message. Callback is invoked with either the response
	// or an error.  See:
	//   https://developer.chrome.com/docs/extensions/reference/runtime/#method-sendMessage
	Send(msg js.Value, callback func(rsp js.Value, err error))
}

// ExtSender sends messages to a single extension.
//
// ExtSender implements the Sender interface.
type ExtSender struct {
	// extensionID is the unique ID for the target extension.
	extensionID string
}

// NewLocalSender returns a ExtSender for sending messages within our own
// extension.
func NewLocalSender() *ExtSender {
	return &ExtSender{
		extensionID: chrome.ExtensionID(),
	}
}

// Send implements Sender.Send().
func (e *ExtSender) Send(msg js.Value, callback func(rsp js.Value, err error)) {
	chrome.Runtime().Call(
		"sendMessage", e.extensionID, msg, nil,
		jsutil.OneTimeFuncOf(func(this js.Value, args []js.Value) interface{} {
			if err := chrome.LastError(); err != nil {
				callback(js.Undefined(), err)
				return nil
			}
			callback(jsutil.SingleArg(args), nil)
			return nil
		}))
}
