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

package keys

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
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

// sendErrorResponse sends a generic error response to the client. This is used
// in case a more specific error is not possible.
func (s *Server) sendErrorResponse(err error, sendResponse func(js.Value)) {
	jsutil.LogError("Server.sendErrorResponse: %v", err)
	rsp := rspError{
		Type: msgTypeErrorRsp,
		Err:  makeErrStr(err),
	}
	sendResponse(vert.ValueOf(rsp).JSValue())
}

// OnMessage is the callback invoked when a message is received. It determines
// the type of request received, invokes the appropriate method on the
// underlying manager instance, and then sends a response with the result.
//
// This method is guaranteed to invoke sendReponse (aside from unexpected
// panics). Context for why this important:
//
//   The caller is expected to be handling an OnMessage event from the browser,
//   and it returns 'true' to the browser to indicate that the event will be
//   handled asynchronously and the port must not yet be closed. Invoking
//   sendResponse is the signal to the browser to close the port and free
//   resources.
func (s *Server) OnMessage(headerObj js.Value, sender js.Value, sendResponse func(js.Value)) {
	var header msgHeader
	if err := vert.ValueOf(headerObj).AssignTo(&header); err != nil {
		s.sendErrorResponse(fmt.Errorf("failed to parse message header: %v", err), sendResponse)
		return
	}

	jsutil.LogDebug("Server.OnMessage(type = %d)", header.Type)
	switch header.Type {
	case msgTypeConfigured:
		jsutil.LogDebug("Server.OnMessage(Configured req)")
		s.mgr.Configured(func(keys []*ConfiguredKey, err error) {
			jsutil.LogDebug("Server.OnMessage(Configured rsp): %d keys, err=%v", len(keys), err)
			rsp := rspConfigured{
				Type: msgTypeConfiguredRsp,
				Keys: keys,
				Err:  makeErrStr(err),
			}
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	case msgTypeLoaded:
		jsutil.LogDebug("Server.OnMessage(Loaded req)")
		s.mgr.Loaded(func(keys []*LoadedKey, err error) {
			jsutil.LogDebug("Server.OnMessage(Loaded rsp): %d keys, err=%v", len(keys), err)
			rsp := rspLoaded{
				Type: msgTypeLoadedRsp,
				Keys: keys,
				Err:  makeErrStr(err),
			}
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	case msgTypeAdd:
		var m msgAdd
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			s.sendErrorResponse(fmt.Errorf("failed to parse Add message: %v", err), sendResponse)
			return
		}
		jsutil.LogDebug("Server.OnMessage(Add req): name=%s", m.Name)
		s.mgr.Add(m.Name, m.PEMPrivateKey, func(err error) {
			rsp := rspAdd{
				Type: msgTypeAddRsp,
				Err:  makeErrStr(err),
			}
			jsutil.LogDebug("Server.OnMessage(Add rsp): err=%v", err)
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	case msgTypeRemove:
		var m msgRemove
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			s.sendErrorResponse(fmt.Errorf("failed to parse Remove message: %v", err), sendResponse)
			return
		}
		jsutil.LogDebug("Server.OnMessage(Remove req): id=%s", m.ID)
		s.mgr.Remove(ID(m.ID), func(err error) {
			rsp := rspRemove{
				Type: msgTypeRemoveRsp,
				Err:  makeErrStr(err),
			}
			jsutil.LogDebug("Server.OnMessage(Remove rsp): err=%v", err)
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	case msgTypeLoad:
		var m msgLoad
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			s.sendErrorResponse(fmt.Errorf("failed to parse Load message: %v", err), sendResponse)
			return
		}
		jsutil.LogDebug("Server.OnMessage(Load req): id=%s", m.ID)
		s.mgr.Load(ID(m.ID), m.Passphrase, func(err error) {
			rsp := rspLoad{
				Type: msgTypeLoadRsp,
				Err:  makeErrStr(err),
			}
			jsutil.LogDebug("Server.OnMessage(Load rsp): err=%v", err)
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	case msgTypeUnload:
		var m msgUnload
		if err := vert.ValueOf(headerObj).AssignTo(&m); err != nil {
			s.sendErrorResponse(fmt.Errorf("failed to parse Unload message: %v", err), sendResponse)
			return
		}
		jsutil.LogDebug("Server.OnMessage(Unload req): id=%s", m.ID)
		s.mgr.Unload(ID(m.ID), func(err error) {
			rsp := rspUnload{
				Type: msgTypeUnloadRsp,
				Err:  makeErrStr(err),
			}
			jsutil.LogDebug("Server.OnMessage(Unload rsp): err=%v", err)
			sendResponse(vert.ValueOf(rsp).JSValue())
		})
		return
	default:
		s.sendErrorResponse(fmt.Errorf("received invalid message type: %d", header.Type), sendResponse)
		return
	}
}

// MessageSender defines methods sufficient to send messages.
type MessageSender interface {
	SendMessage(msg js.Value, callback func(rsp js.Value, err error))
}

// client implements the Manager interface and forwards calls to a Server.
type client struct {
	msg MessageSender
}

// NewClient returns a Manager implementation that forwards calls to a Server.
func NewClient(msg MessageSender) Manager {
	return &client{msg: msg}
}

// Configured implements Manager.Configured.
func (c *client) Configured(callback func(keys []*ConfiguredKey, err error)) {
	var msg msgConfigured
	msg.Type = msgTypeConfigured
	jsutil.LogDebug("Client.Configured(req)")
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Configured(rsp)")
		if err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspConfigured
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(nil, fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

// Loaded implements Manager.Loaded.
func (c *client) Loaded(callback func(keys []*LoadedKey, err error)) {
	var msg msgLoaded
	msg.Type = msgTypeLoaded
	jsutil.LogDebug("Client.Loaded(req)")
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Loaded(rsp)")
		if err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspLoaded
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(nil, fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

// Add implements Manager.Add.
func (c *client) Add(name string, pemPrivateKey string, callback func(err error)) {
	var msg msgAdd
	msg.Type = msgTypeAdd
	msg.Name = name
	msg.PEMPrivateKey = pemPrivateKey
	jsutil.LogDebug("Client.Add(req): name=%s", msg.Name)
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Add(rsp)")
		if err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspAdd
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Remove implements Manager.Remove.
func (c *client) Remove(id ID, callback func(err error)) {
	var msg msgRemove
	msg.Type = msgTypeRemove
	msg.ID = string(id)
	jsutil.LogDebug("Client.Remove(req): id=%s", msg.ID)
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Remove(rsp)")
		if err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspRemove
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Load implements Manager.Load.
func (c *client) Load(id ID, passphrase string, callback func(err error)) {
	var msg msgLoad
	msg.Type = msgTypeLoad
	msg.ID = string(id)
	msg.Passphrase = passphrase
	jsutil.LogDebug("Client.Load(req): id=%s", msg.ID)
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Load(rsp)")
		if err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspLoad
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Unload implements Manager.Unload.
func (c *client) Unload(id ID, callback func(err error)) {
	var msg msgUnload
	msg.Type = msgTypeUnload
	msg.ID = string(id)
	jsutil.LogDebug("Client.Unload(req): id=%s", msg.ID)
	c.msg.SendMessage(vert.ValueOf(msg).JSValue(), func(rspObj js.Value, err error) {
		jsutil.LogDebug("Client.Unload(rsp)")
		if err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		var rsp rspUnload
		if err := vert.ValueOf(rspObj).AssignTo(&rsp); err != nil {
			callback(fmt.Errorf("failed to parse response: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}
