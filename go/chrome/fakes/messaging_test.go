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

package fakes

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

func TestMessagePassing(t *testing.T) {
	hub := NewMessageHub()

	// Add handlers that respond to different values.
	hub.OnMessage(func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool {
		if header.Int() == 42 {
			sendResponse("int")
		}
		return true
	})
	hub.OnMessage(func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool {
		if header.String() == "foo" {
			sendResponse("string")
		}
		return true
	})
	hub.OnMessage(func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool {
		if header.Get("some-key") != js.Undefined {
			sendResponse("map")
		}
		return true
	})

	// Send messages of the various types.
	var intRsp, strRsp, mapRsp *js.Object
	hub.SendMessage(42, func(rsp *js.Object) {
		intRsp = rsp
	})
	hub.SendMessage("foo", func(rsp *js.Object) {
		strRsp = rsp
	})
	hub.SendMessage(map[string]int{"some-key": 7}, func(rsp *js.Object) {
		mapRsp = rsp
	})

	// Ensure we got the correct responses.
	if diff := pretty.Diff(intRsp.String(), "int"); diff != nil {
		t.Errorf("incorrect response for int; -got +want: %s", diff)
	}
	if diff := pretty.Diff(strRsp.String(), "string"); diff != nil {
		t.Errorf("incorrect response for string; -got +want: %s", diff)
	}
	if diff := pretty.Diff(mapRsp.String(), "map"); diff != nil {
		t.Errorf("incorrect response for map; -got +want: %s", diff)
	}
}
