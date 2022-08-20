//go:build js

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
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/storage"
	"golang.org/x/crypto/ssh/agent"
)

var (
	// Create a keyring with loaded keys.
	agt = agent.NewKeyring()

	// Create a wrapper that can update the loaded keys. Exposed the
	// wrapper so it can be used by other pages in the extension.
	mgr = keys.NewManager(agt, storage.DefaultSync(), storage.DefaultSession())
	svr = keys.NewServer(mgr)

	// Reload any keys for the session into the agent.
	_ = func() bool {
		mgr.LoadFromSession(func(err error) {
			if err != nil {
				jsutil.LogError("failed to load keys into agent: %v", err)
			}
		})
		return true // Dummy value.
	}()

	// Keep a set of ports that are open for communicating between
	// clients and agents.
	ports = agentport.AgentPorts{}
)

func onMessage(this js.Value, args []js.Value) interface{} {
	var message, sender, sendResponse js.Value
	jsutil.ExpandArgs(args, &message, &sender, &sendResponse)
	svr.OnMessage(message, sender, func(rsp js.Value) {
		sendResponse.Invoke(rsp)
	})
	return nil
}

func onConnectExternal(this js.Value, args []js.Value) interface{} {
	port := jsutil.SingleArg(args)
	if ports.Lookup(port) != nil {
		jsutil.LogError("onConnectExternal: port already in use; ignoring")
		return nil
	}

	jsutil.LogDebug("onConnectExternal: connecting new port")
	ap := agentport.New(port)
	ports.Add(port, ap)

	jsutil.LogDebug("onConnectExternal: serving in background")
	go func() {
		jsutil.LogDebug("ServeAgent: starting for new port")
		defer jsutil.LogDebug("ServeAgent: finished")
		if err := agent.ServeAgent(agt, ap); err != nil {
			jsutil.LogDebug("ServeAgent: finished with error: %v", err)
		}
	}()
	return nil
}

func onConnectionMessage(this js.Value, args []js.Value) interface{} {
	var port, msg js.Value
	jsutil.ExpandArgs(args, &port, &msg)

	ap := ports.Lookup(port)
	if ap == nil {
		jsutil.LogError("onConnectionMessage: connection for port not found; ignoring")
		return nil
	}

	jsutil.LogDebug("onConnectionMessage: forwarding message")
	ap.OnMessage(msg)
	return nil
}

func onConnectionDisconnect(this js.Value, args []js.Value) interface{} {
	port := jsutil.SingleArg(args)

	ap := ports.Lookup(port)
	if ap == nil {
		jsutil.LogError("onConnectionDisconnect: connection for port not found; ignoring")
		return nil
	}

	jsutil.LogDebug("onConnectionDisconnect: disconnecting")
	ap.OnDisconnect()
	ports.Delete(port)
	return nil
}

func main() {
	jsutil.Log("Starting background worker")
	defer jsutil.Log("Exiting background worker")

	c1 := jsutil.DefineFunc(js.Global(), "handleOnMessage", onMessage)
	defer c1()
	c2 := jsutil.DefineFunc(js.Global(), "handleOnConnectExternal", onConnectExternal)
	defer c2()
	c3 := jsutil.DefineFunc(js.Global(), "handleConnectionMessage", onConnectionMessage)
	defer c3()
	c4 := jsutil.DefineFunc(js.Global(), "handleConnectionDisconnect", onConnectionDisconnect)
	defer c4()

	done := make(chan struct{}, 0)
	<-done // Do not terminate immediately.
}
