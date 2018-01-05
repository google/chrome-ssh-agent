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

	"github.com/google/chrome-ssh-agent/go/chrome"
	"github.com/gopherjs/gopherjs/js"
)

type Server struct {
	mgr    Manager
	chrome *chrome.C
}

func NewServer(mgr Manager, chrome *chrome.C) *Server {
	result := &Server{
		mgr:    mgr,
		chrome: chrome,
	}
	result.chrome.OnMessage(result.onMessage)
	return result
}

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
)

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
	Id ID `js:"id"`
}

type rspRemove struct {
	*msgHeader
	Err string `js:"err"`
}

type msgLoad struct {
	*msgHeader
	Id         ID     `js:"id"`
	Passphrase string `js:"passphrase"`
}

type rspLoad struct {
	*msgHeader
	Err string `js:"err"`
}

func makeErr(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func makeErrStr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

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
		s.mgr.Remove(m.Id, func(err error) {
			rsp := &rspRemove{msgHeader: header}
			rsp.Type = msgTypeRemoveRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	case msgTypeLoad:
		m := &msgLoad{msgHeader: header}
		s.mgr.Load(m.Id, m.Passphrase, func(err error) {
			rsp := &rspLoad{msgHeader: header}
			rsp.Type = msgTypeLoadRsp
			rsp.Err = makeErrStr(err)
			sendResponse(rsp)
		})
	}
	return true
}

type client struct {
	chrome *chrome.C
}

func NewClient(chrome *chrome.C) Manager {
	return &client{chrome: chrome}
}

func (c *client) Configured(callback func(keys []*ConfiguredKey, err error)) {
	msg := &msgConfigured{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeConfigured
	c.chrome.SendMessage(c.chrome.ExtensionId(), msg, func(rspObj *js.Object) {
		rsp := &rspConfigured{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.chrome.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

func (c *client) Loaded(callback func(keys []*LoadedKey, err error)) {
	msg := &msgLoaded{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoaded
	c.chrome.SendMessage(c.chrome.ExtensionId(), msg, func(rspObj *js.Object) {
		rsp := &rspLoaded{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.chrome.Error(); err != nil {
			callback(nil, fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(rsp.Keys, makeErr(rsp.Err))
	})
}

func (c *client) Add(name string, pemPrivateKey string, callback func(err error)) {
	msg := &msgAdd{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeAdd
	msg.Name = name
	msg.PEMPrivateKey = pemPrivateKey
	c.chrome.SendMessage(c.chrome.ExtensionId(), msg, func(rspObj *js.Object) {
		rsp := &rspAdd{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

func (c *client) Remove(id ID, callback func(err error)) {
	msg := &msgRemove{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeRemove
	msg.Id = id
	c.chrome.SendMessage(c.chrome.ExtensionId(), msg, func(rspObj *js.Object) {
		rsp := &rspRemove{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}

func (c *client) Load(id ID, passphrase string, callback func(err error)) {
	msg := &msgLoad{msgHeader: &msgHeader{Object: js.Global.Get("Object").New()}}
	msg.Type = msgTypeLoad
	msg.Id = id
	msg.Passphrase = passphrase
	c.chrome.SendMessage(c.chrome.ExtensionId(), msg, func(rspObj *js.Object) {
		rsp := &rspLoad{msgHeader: &msgHeader{Object: rspObj}}
		if err := c.chrome.Error(); err != nil {
			callback(fmt.Errorf("failed to send message: %v", err))
			return
		}
		callback(makeErr(rsp.Err))
	})
}
