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

	"github.com/gopherjs/gopherjs/js"
)

// MessageReceiver defines methods sufficient to receive messages and send
// responses.
type MessageReceiver interface {
	OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool)
}

// Server exposes a Manager instance via a messaging API so that a shared
// instance can be invoked from a different page.
type Server struct {
	mgr Manager
	msg MessageReceiver
}

// NewServer returns a new Server that manages keys using the
// supplied Manager.
func NewServer(mgr Manager, msg MessageReceiver) *Server {
	result := &Server{
		mgr: mgr,
		msg: msg,
	}
	result.msg.OnMessage(result.onMessage)
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
)

// msgHeader are the common fields included in every message (as an embedded
// type).
type msgHeader struct {
	*js.Object
	Type int `js:"type"`
}

type msgConfigured struct {
	*msgHeader
}

type rspConfigured struct {
	*msgHeader
	Keys []*ConfiguredKey `js:"keys"`
	Err  string           `js:"err"`
}

type msgLoaded struct {
	*msgHeader
}

type rspLoaded struct {
	*msgHeader
	Keys []*LoadedKey `js:"keys"`
	Err  string       `js:"err"`
}

type msgAdd struct {
	*msgHeader
	Name          string `js:"name"`
	PEMPrivateKey string `js:"pemPrivateKey"`
}

type rspAdd struct {
	*msgHeader
	Err string `js:"err"`
}

type msgRemove struct {
	*msgHeader
	ID ID `js:"id"`
}

type rspRemove struct {
	*msgHeader
	Err string `js:"err"`
}

type msgLoad struct {
	*msgHeader
	ID         ID     `js:"id"`
	Passphrase string `js:"passphrase"`
}

type rspLoad struct {
	*msgHeader
	Err string `js:"err"`
}

type msgUnload struct {
	*msgHeader
	Key *LoadedKey `js:"key"`
}

type rspUnload struct {
	*msgHeader
	Err string `js:"err"`
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

// onMessage is the callback invoked when a message is received. It determines
// the type of request received, invokes the appropriate method on the
// underlying manager instance, and then sends a response with the result.
func (s *Server) onMessage(headerObj *js.Object, sender *js.Object, sendResponse func(interface{})) bool {
	header := &msgHeader{Object: headerObj}
	switch header.Type {
	case msgTypeConfigured:
		s.mgr.Configured(func(keys []*ConfiguredKey, err error) {
			rsp := &rspConfigured{msgHeader: header}
			rsp.Type = msgTypeConfiguredRsp
			rsp.Keys = keys
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeLoaded:
		s.mgr.Loaded(func(keys []*LoadedKey, err error) {
			rsp := &rspLoaded{msgHeader: header}
			rsp.Type = msgTypeLoadedRsp
			rsp.Keys = keys
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeAdd:
		m := &msgAdd{msgHeader: header}
		s.mgr.Add(m.Name, m.PEMPrivateKey, func(err error) {
			rsp := &rspAdd{msgHeader: header}
			rsp.Type = msgTypeAddRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeRemove:
		m := &msgRemove{msgHeader: header}
		s.mgr.Remove(m.ID, func(err error) {
			rsp := &rspRemove{msgHeader: header}
			rsp.Type = msgTypeRemoveRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeLoad:
		m := &msgLoad{msgHeader: header}
		s.mgr.Load(m.ID, m.Passphrase, func(err error) {
			rsp := &rspLoad{msgHeader: header}
			rsp.Type = msgTypeLoadRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeUnload:
		m := &msgUnload{msgHeader: header}
		s.mgr.Unload(m.Key, func(err error) {
			rsp := &rspUnload{msgHeader: header}
			rsp.Type = msgTypeUnloadRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	}
	return true
}

// MessageSender defines methods sufficient to send messages.
type MessageSender interface {
	SendMessage(msg interface{}, callback func(rsp *js.Object))
	Error() error
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
	msg := &msgConfigured{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeConfigured
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspConfigured{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

// Loaded implements Manager.Loaded.
func (c *client) Loaded(callback func(keys []*LoadedKey, err error)) {
	msg := &msgLoaded{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoaded
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspLoaded{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

// Add implements Manager.Add.
func (c *client) Add(name string, pemPrivateKey string, callback func(err error)) {
	msg := &msgAdd{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeAdd
	msg.Name = name
	msg.PEMPrivateKey = pemPrivateKey
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspAdd{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Remove implements Manager.Remove.
func (c *client) Remove(id ID, callback func(err error)) {
	msg := &msgRemove{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeRemove
	msg.ID = id
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspRemove{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Load implements Manager.Load.
func (c *client) Load(id ID, passphrase string, callback func(err error)) {
	msg := &msgLoad{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoad
	msg.ID = id
	msg.Passphrase = passphrase
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspLoad{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

// Unload implements Manager.Unload.
func (c *client) Unload(key *LoadedKey, callback func(err error)) {
	msg := &msgUnload{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeUnload
	msg.Key = key
	c.msg.SendMessage(msg, func(rspObj *js.Object) {
		rsp := &rspUnload{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.msg.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}
