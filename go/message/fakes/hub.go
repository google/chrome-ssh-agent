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

package fakes

import (
	"syscall/js"
)

// Receiver defines methods sufficient to receive messages and send
// responses.
type Receiver interface {
	OnMessage(header js.Value, sender js.Value, sendResponse func(js.Value))
}

// Hub is a fake implementation of Chrome's messaging APIs.
type Hub struct {
	receivers []Receiver
}

// NewHub returns a fake implementation of Chrome's messaging APIs.
func NewHub() *Hub {
	return &Hub{}
}

// AddReceiver adds a receiver to which messages should be delivered.
func (m *Hub) AddReceiver(r Receiver) {
	m.receivers = append(m.receivers, r)
}

// Send implements Sender.Send().
func (m *Hub) Send(msg js.Value, callback func(rsp js.Value, err error)) {
	for _, r := range m.receivers {
		r.OnMessage(msg, js.Null(), func(rsp js.Value) {
			callback(rsp, nil)
		})
	}
}
