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

package keys

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/message"
	"github.com/norunners/vert"
)

// Server exposes a Manager instance via a messaging API so that a shared
// instance can be invoked from a different page.
type Server struct {
	mgr Manager
}

// NewServer returns a new Server that manages keys using the
// supplied Manager.
func NewServer(mgr Manager) *Server {
	result := &Server{
		mgr: mgr,
	}
	return result
}

// Define a distinct type for each message.  These are embedded in each
// message.
const (
	msgTypeConfigured int = 1000 + iota
	msgTypeConfiguredRsp
	msgTypeLoaded
	msgTypeLoadedRsp
	msgTypeAdd
	msgTypeAddRsp
	msgTypeRemove
	msgTypeRemoveRsp
	msgTypeLoad
	msgTypeLoadRsp
	msgTypeUnload
	msgTypeUnloadRsp
	msgTypeErrorRsp
)

// msgHeader are the common fields included in every message.
type msgHeader struct {
	Type int `js:"type"`
}

type msgConfigured struct {
	Type int `js:"type"`
}

type rspConfigured struct {
	Type int              `js:"type"`
	Keys []*ConfiguredKey `js:"keys"`
	Err  string           `js:"err"`
}

type msgLoaded struct {
	Type int `js:"type"`
}

type rspLoaded struct {
	Type int          `js:"type"`
	Keys []*LoadedKey `js:"keys"`
	Err  string       `js:"err"`
}

type msgAdd struct {
	Type          int    `js:"type"`
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

type rspAdd struct {
	Type int    `js:"type"`
	Err  string `js:"err"`
}

type msgRemove struct {
	Type int    `js:"type"`
	ID   string `js:"id"`
}

type rspRemove struct {
	Type int    `js:"type"`
	Err  string `js:"err"`
}

type msgLoad struct {
	Type       int    `js:"type"`
	ID         string `js:"id"`
	Passphrase string `js:"passphrase"`
}

type rspLoad struct {
	Type int    `js:"type"`
	Err  string `js:"err"`
}

type msgUnload struct {
	Type int    `js:"type"`
	ID   string `js:"id"`
}

type rspUnload struct {
	Type int    `js:"type"`
	Err  string `js:"err"`
}

type rspError struct {
	Type int    `js:"type"`
	Err  string `js:"err"`
}

// makeErr converts a string to an error. Empty string returns nil (i.e., no
// error).
func makeErr(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

// makeErrStr converts an error to a string. A nil error is converted to the
// empty string.
func makeErrStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// makeErrorResponse produces a generic error response that can be sent to the
// client. This is used in case a more specific error is not possible.
func (s *Server) makeErrorResponse(err error) js.Value {
	jsutil.LogError("Server.makeErrorResponse: %v", err)
	rsp := rspError{
		Type: msgTypeErrorRsp,
		Err:  makeErrStr(err),
	}
	return vert.ValueOf(rsp).JSValue()
}

// OnMessage is the callback invoked when a message is received. It determines
// the type of request received, invokes the appropriate method on the
// underlying manager instance, and then returns the response to be sent to the
// client.
func (s *Server) OnMessage(ctx jsutil.AsyncContext, headerObj js.Value, sender js.Value) js.Value {
	var header msgHeader
	if err := vert.ValueOf(headerObj).AssignTo(&header); err != nil {
		return s.makeErrorResponse(fmt.Errorf("failed to parse message header: %v", err))
	}

	jsutil.LogDebug("Server.OnMessage(type = %d)", header.Type)
	switch header.Type {
	case msgTypeConfigured:
		jsutil.LogDebug("Server.OnMessage(Configured req)")
		keys, err := s.mgr.Configured(ctx)
		jsutil.LogDebug("Server.OnMessage(Configured rsp): %d keys, err=%v", len(keys), err)
		rsp := rspConfigured{
			Type: msgTypeConfiguredRsp,
			Keys: keys,
			Err:  makeErrStr(err),
		}
		return vert.ValueOf(rsp).JSValue()
	case msgTypeLoaded:
		jsutil.LogDebug("Server.OnMessage(Loaded req)")
		keys, err := s.mgr.Loaded(ctx)
		jsutil.LogDebug("Server.OnMessage(Loaded rsp): %d keys, err=%v", len(keys), err)
		rsp := rspLoaded{
			Type: msgTypeLoadedRsp,
			Keys: keys,
			Err:  makeErrStr(err),
		}
		return vert.ValueOf(rsp).JSValue()
	case msgTypeAdd:
		var m msgAdd
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			return s.makeErrorResponse(fmt.Errorf("failed to parse Add message: %v", err))
		}
		jsutil.LogDebug("Server.OnMessage(Add req): name=%s", m.Name)
		err := s.mgr.Add(ctx, m.Name, m.PEMPrivateKey)
		rsp := rspAdd{
			Type: msgTypeAddRsp,
			Err:  makeErrStr(err),
		}
		jsutil.LogDebug("Server.OnMessage(Add rsp): err=%v", err)
		return vert.ValueOf(rsp).JSValue()
	case msgTypeRemove:
		var m msgRemove
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			return s.makeErrorResponse(fmt.Errorf("failed to parse Remove message: %v", err))
		}
		jsutil.LogDebug("Server.OnMessage(Remove req): id=%s", m.ID)
		err := s.mgr.Remove(ctx, ID(m.ID))
		rsp := rspRemove{
			Type: msgTypeRemoveRsp,
			Err:  makeErrStr(err),
		}
		jsutil.LogDebug("Server.OnMessage(Remove rsp): err=%v", err)
		return vert.ValueOf(rsp).JSValue()
	case msgTypeLoad:
		var m msgLoad
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			return s.makeErrorResponse(fmt.Errorf("failed to parse Load message: %v", err))
		}
		jsutil.LogDebug("Server.OnMessage(Load req): id=%s", m.ID)
		err := s.mgr.Load(ctx, ID(m.ID), m.Passphrase)
		rsp := rspLoad{
			Type: msgTypeLoadRsp,
			Err:  makeErrStr(err),
		}
		jsutil.LogDebug("Server.OnMessage(Load rsp): err=%v", err)
		return vert.ValueOf(rsp).JSValue()
	case msgTypeUnload:
		var m msgUnload
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			return s.makeErrorResponse(fmt.Errorf("failed to parse Unload message: %v", err))
		}
		jsutil.LogDebug("Server.OnMessage(Unload req): id=%s", m.ID)
		err := s.mgr.Unload(ctx, ID(m.ID))
		rsp := rspUnload{
			Type: msgTypeUnloadRsp,
			Err:  makeErrStr(err),
		}
		jsutil.LogDebug("Server.OnMessage(Unload rsp): err=%v", err)
		return vert.ValueOf(rsp).JSValue()
	default:
		return s.makeErrorResponse(fmt.Errorf("received invalid message type: %d", header.Type))
	}
}

// client implements the Manager interface and forwards calls to a Server.
type client struct {
	msg message.Sender
}

// NewClient returns a Manager implementation that forwards calls to a Server.
func NewClient(msg message.Sender) Manager {
	return &client{msg: msg}
}

// Configured implements Manager.Configured.
func (c *client) Configured(ctx jsutil.AsyncContext) ([]*ConfiguredKey, error) {
	var msg msgConfigured
	msg.Type = msgTypeConfigured
	jsutil.LogDebug("Client.Configured(req)")
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Configured(rsp)")
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspConfigured
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	return rsp.Keys, makeErr(rsp.Err)
}

// Loaded implements Manager.Loaded.
func (c *client) Loaded(ctx jsutil.AsyncContext) ([]*LoadedKey, error) {
	var msg msgLoaded
	msg.Type = msgTypeLoaded
	jsutil.LogDebug("Client.Loaded(req)")
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Loaded(rsp)")
	if err != nil {
		return nil, fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspLoaded
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	return rsp.Keys, makeErr(rsp.Err)
}

// Add implements Manager.Add.
func (c *client) Add(ctx jsutil.AsyncContext, name string, pemPrivateKey string) error {
	var msg msgAdd
	msg.Type = msgTypeAdd
	msg.Name = name
	msg.PEMPrivateKey = pemPrivateKey
	jsutil.LogDebug("Client.Add(req): name=%s", msg.Name)
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Add(rsp)")
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspAdd
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	return makeErr(rsp.Err)
}

// Remove implements Manager.Remove.
func (c *client) Remove(ctx jsutil.AsyncContext, id ID) error {
	var msg msgRemove
	msg.Type = msgTypeRemove
	msg.ID = string(id)
	jsutil.LogDebug("Client.Remove(req): id=%s", msg.ID)
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Remove(rsp)")
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspRemove
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	return makeErr(rsp.Err)
}

// Load implements Manager.Load.
func (c *client) Load(ctx jsutil.AsyncContext, id ID, passphrase string) error {
	var msg msgLoad
	msg.Type = msgTypeLoad
	msg.ID = string(id)
	msg.Passphrase = passphrase
	jsutil.LogDebug("Client.Load(req): id=%s", msg.ID)
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Load(rsp)")
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspLoad
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	return makeErr(rsp.Err)
}

// Unload implements Manager.Unload.
func (c *client) Unload(ctx jsutil.AsyncContext, id ID) error {
	var msg msgUnload
	msg.Type = msgTypeUnload
	msg.ID = string(id)
	jsutil.LogDebug("Client.Unload(req): id=%s", msg.ID)
	rspObj, err := c.msg.Send(ctx, vert.ValueOf(msg).JSValue())
	jsutil.LogDebug("Client.Unload(rsp)")
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	var rsp rspUnload
	if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	return makeErr(rsp.Err)
}
