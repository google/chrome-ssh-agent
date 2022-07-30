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

package main

import (
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/agentport"
	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"golang.org/x/crypto/ssh/agent"
)

func main() {
	dom.Log("Starting background worker")
	defer dom.Log("Exiting background worker")

	done := make(chan struct{}, 0)

	// Create a keyring with loaded keys.
	a := agent.NewKeyring()

	// Create a wrapper that can update the loaded keys. Exposed the
	// wrapper so it can be used by other pages in the extension.
	c := chrome.New(js.Null())
	mgr := keys.NewManager(a, c.SyncStorage())
	keys.NewServer(mgr, c)

	c.OnConnectExternal(func(port js.Value) {
		dom.Log("Starting agent for new port")
		go agent.ServeAgent(a, agentport.New(port))
	})

	<-done // Do not terminate.
}
