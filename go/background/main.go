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
	"errors"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/agentport"
	"github.com/google/chrome-ssh-agent/go/app"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/storage"
	"golang.org/x/crypto/ssh/agent"
)

type background struct {
	// agent is keyring with the loaded keys.
	agent agent.Agent
	// ports manages opened ports for communicating with the agent.
	ports agentport.AgentPorts
	// manager is a wrapper that can manage loaded keys.
	manager *keys.DefaultManager
	// server exposes an API for the manager.
	server *keys.Server
}

func newBackground() *background {
	agt := agent.NewKeyring()
	mgr := keys.NewManager(agt, storage.DefaultSync(), storage.DefaultSession())
	return &background{
		agent:   agt,
		ports:   agentport.AgentPorts{},
		manager: mgr,
		server:  keys.NewServer(mgr),
	}
}

func (a *background) Name() string {
	return "BackgroundWorker"
}

func (a *background) Init(ctx jsutil.AsyncContext, cleanup *jsutil.CleanupFuncs) error {
	jsutil.Log("Initializing manager")
	if err := a.manager.Init(ctx); err != nil {
		jsutil.LogError("failed to initialize manager: %v", err)
	}

	jsutil.LogDebug("Attaching event handlers")
	cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleOnMessage", a.onMessage))
	cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleConnectionMessage", a.onConnectionMessage))
	cleanup.Add(jsutil.DefineAsyncFunc(js.Global(), "handleConnectionDisconnect", a.onConnectionDisconnect))
	return nil
}

func (a *background) onMessage(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	var message, sender, sendResponse js.Value
	jsutil.ExpandArgs(args, &message, &sender, &sendResponse)
	rsp := a.server.OnMessage(ctx, message, sender)
	sendResponse.Invoke(rsp)
	return js.Undefined(), nil
}

func (a *background) addPort(port js.Value) *agentport.AgentPort {
	ap := agentport.New(port)
	a.ports.Add(port, ap)

	go func() {
		jsutil.LogDebug("ServeAgent: starting for new port")
		defer jsutil.LogDebug("ServeAgent: finished")
		if err := agent.ServeAgent(a.agent, ap); err != nil {
			jsutil.LogDebug("ServeAgent: finished with error: %v", err)
		}
	}()

	return ap
}

func (a *background) onConnectionMessage(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	var port, msg js.Value
	jsutil.ExpandArgs(args, &port, &msg)

	ap := a.ports.Lookup(port)
	if ap == nil {
		// We spawn a new connection on-demand when we notice a new port.
		// While a typical place to do this would have been in an
		// OnConnectExternal event handler, both were asynchronously
		// executed (in our model, anyways) and we don't have any
		// guarantee that it will happen prior to receiving the first
		// message.
		jsutil.LogDebug("onConnectionMessage: existing connection not found; spawning")
		ap = a.addPort(port)
	}

	jsutil.LogDebug("onConnectionMessage: forwarding message")
	ap.OnMessage(msg)
	return js.Undefined(), nil
}

func (a *background) onConnectionDisconnect(ctx jsutil.AsyncContext, this js.Value, args []js.Value) (js.Value, error) {
	port := jsutil.SingleArg(args)

	ap := a.ports.Lookup(port)
	if ap == nil {
		err := errors.New("onConnectionDisconnect: connection for port not found")
		jsutil.LogError(err.Error())
		return js.Undefined(), err
	}

	jsutil.LogDebug("onConnectionDisconnect: disconnecting")
	ap.OnDisconnect()
	a.ports.Delete(port)
	return js.Undefined(), nil
}

func main() {
	a := app.New(newBackground())
	defer a.Release()
	a.Run()
}
