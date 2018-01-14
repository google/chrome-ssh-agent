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

package keys

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

type dummyManager struct {
	ID             ID
	Name           string
	PEMPrivateKey  string
	Passphrase     string
	ConfiguredKeys []*ConfiguredKey
	LoadedKeys     []*LoadedKey
	Key            *LoadedKey
	Err            error
}

func (m *dummyManager) Configured(callback func(keys []*ConfiguredKey, err error)) {
	callback(m.ConfiguredKeys, m.Err)
}

func (m *dummyManager) Add(name string, pemPrivateKey string, callback func(err error)) {
	m.Name = name
	m.PEMPrivateKey = pemPrivateKey
	callback(m.Err)
}

func (m *dummyManager) Remove(id ID, callback func(err error)) {
	m.ID = id
	callback(m.Err)
}

func (m *dummyManager) Loaded(callback func(keys []*LoadedKey, err error)) {
	callback(m.LoadedKeys, m.Err)
}

func (m *dummyManager) Load(id ID, passphrase string, callback func(err error)) {
	m.ID = id
	m.Passphrase = passphrase
	callback(m.Err)
}

func (m *dummyManager) Unload(key *LoadedKey, callback func(err error)) {
	m.Key = key
	callback(m.Err)
}

func TestClientServerConfigured(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	k0 := &ConfiguredKey{Object: js.Global.Get("Object").New()}
	k0.ID = ID("id-0")
	k0.Name = "key-0"
	k1 := &ConfiguredKey{Object: js.Global.Get("Object").New()}
	k1.ID = ID("id-1")
	k1.Name = "key-1"

	wantConfiguredKeys := []*ConfiguredKey{k0, k1}
	wantErr := errors.New("failed")

	mgr.ConfiguredKeys = append(mgr.ConfiguredKeys, wantConfiguredKeys...)
	mgr.Err = wantErr

	configured, err := syncConfigured(cli)
	// Compare using reflect.DeepEqual since pretty.Diff fails to
	// terminate on this input.
	if !reflect.DeepEqual(configured, wantConfiguredKeys) {
		t.Errorf("incorrect configured keys; got %v, want %v", configured, wantConfiguredKeys)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerAdd(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantName := "some-name"
	wantPrivateKey := "private-key"
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncAdd(cli, wantName, wantPrivateKey)
	if diff := pretty.Diff(mgr.Name, wantName); diff != nil {
		t.Errorf("incorrect name; -got +want: %s", diff)
	}
	if diff := pretty.Diff(mgr.PEMPrivateKey, wantPrivateKey); diff != nil {
		t.Errorf("incorrect private key; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerRemove(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantID := ID("id-0")
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncRemove(cli, wantID)
	if diff := pretty.Diff(mgr.ID, wantID); diff != nil {
		t.Errorf("incorrect ID; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerLoaded(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	k0 := &LoadedKey{Object: js.Global.Get("Object").New()}
	k0.Type = "type-0"
	k0.SetBlob([]byte("blob-0"))
	k0.Comment = "comment-0"
	k1 := &LoadedKey{Object: js.Global.Get("Object").New()}
	k1.Type = "type-1"
	k1.SetBlob([]byte("blob-1"))
	k1.Comment = "comment-1"

	wantLoadedKeys := []*LoadedKey{k0, k1}
	wantErr := errors.New("failed")

	mgr.LoadedKeys = append(mgr.LoadedKeys, wantLoadedKeys...)
	mgr.Err = wantErr

	loaded, err := syncLoaded(cli)
	// Compare using reflect.DeepEqual since pretty.Diff fails to
	// terminate on this input.
	if !reflect.DeepEqual(loaded, wantLoadedKeys) {
		t.Errorf("incorrect loaded keys; got %s, want %s", loaded, wantLoadedKeys)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerLoad(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantID := ID("id-0")
	wantPassphrase := "secret"
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncLoad(cli, wantID, wantPassphrase)
	if diff := pretty.Diff(mgr.ID, wantID); diff != nil {
		t.Errorf("incorrect ID; -got +want: %s", diff)
	}
	if diff := pretty.Diff(mgr.Passphrase, wantPassphrase); diff != nil {
		t.Errorf("incorrect passphrase; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerUnload(t *testing.T) {
	hub := fakes.NewMessageHub()
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantKey := &LoadedKey{Object: js.Global.Get("Object").New()}
	wantKey.Type = "type-0"
	wantKey.SetBlob([]byte("blob-0"))
	wantKey.Comment = "comment1"
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncUnload(cli, wantKey)
	// Compare using reflect.DeepEqual since pretty.Diff causes test to fail
	// without any output.
	if !reflect.DeepEqual(mgr.Key, wantKey) {
		t.Errorf("incorrect key; got %s, want %s", mgr.Key, wantKey)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}
