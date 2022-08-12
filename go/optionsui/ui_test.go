//go:build js && wasm

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

package optionsui

import (
	"fmt"
	"syscall/js"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/google/chrome-ssh-agent/go/dom"
	dt "github.com/google/chrome-ssh-agent/go/dom/testing"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	"github.com/google/chrome-ssh-agent/go/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var (
	validID = keys.ID("1")

	// Don't bother with Comment field, since it may contain a
	// randomly-generated ID.
	displayedKeyCmp = cmpopts.IgnoreFields(displayedKey{}, "Comment", "cleanup")

	optionsHTMLData = string(testutil.MustReadRunfile("html/options.html"))
)

type testHarness struct {
	messaging *fakes.MessageHub
	agent     agent.Agent
	manager   keys.Manager
	server    *keys.Server
	Client    keys.Manager
	dom       *dom.DOM
	UI        *UI

	addButton        js.Value
	addName          js.Value
	addKey           js.Value
	addOk            js.Value
	addCancel        js.Value
	passphraseInput  js.Value
	passphraseOk     js.Value
	passphraseCancel js.Value
	removeYes        js.Value
	removeNo         js.Value
}

func (h *testHarness) Release() {
	h.UI.Release()
}

func newHarness() *testHarness {
	syncStorage := fakes.NewMemStorage()
	sessionStorage := fakes.NewMemStorage()
	msg := fakes.NewMessageHub()

	agt := agent.NewKeyring()
	mgr := keys.NewManager(agt, syncStorage, sessionStorage)
	srv := keys.NewServer(mgr)
	msg.AddReceiver(srv)
	cli := keys.NewClient(msg)
	domObj := dom.New(dt.NewDocForTesting(optionsHTMLData))
	ui := New(cli, domObj)

	return &testHarness{
		messaging:        msg,
		agent:            agt,
		manager:          mgr,
		server:           srv,
		Client:           cli,
		dom:              domObj,
		UI:               ui,
		addButton:        domObj.GetElement("add"),
		addName:          domObj.GetElement("addName"),
		addKey:           domObj.GetElement("addKey"),
		addOk:            domObj.GetElement("addOk"),
		addCancel:        domObj.GetElement("addCancel"),
		passphraseInput:  domObj.GetElement("passphrase"),
		passphraseOk:     domObj.GetElement("passphraseOk"),
		passphraseCancel: domObj.GetElement("passphraseCancel"),
		removeYes:        domObj.GetElement("removeYes"),
		removeNo:         domObj.GetElement("removeNo"),
	}
}

func directLoadKey(agt agent.Agent, privateKey string) {
	priv, err := ssh.ParseRawPrivateKey([]byte(privateKey))
	if err != nil {
		panic(fmt.Sprintf("failed to parse private key: %v", err))
	}

	if err := agt.Add(agent.AddedKey{PrivateKey: priv}); err != nil {
		panic(fmt.Sprintf("failed to load private key: %v", err))
	}
}

func findKey(disp []*displayedKey, name string) keys.ID {
	for _, k := range disp {
		if k.Name == name {
			return k.ID
		}
	}
	return keys.InvalidID
}

func equalizeIds(disp []*displayedKey) []*displayedKey {
	var result []*displayedKey
	for _, k := range disp {
		nk := *k
		if nk.ID != keys.InvalidID {
			nk.ID = validID
		}
		result = append(result, &nk)
	}
	return result
}

func TestUserActions(t *testing.T) {
	testcases := []struct {
		description   string
		sequence      func(h *testHarness)
		wantDisplayed []*displayedKey
		wantErr       string
	}{
		{
			description: "add key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, "private-key")
				h.dom.DoClick(h.addOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key",
				},
			},
		},
		{
			description: "add multiple keys",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-1")
				h.dom.SetValue(h.addKey, "private-key-1")
				h.dom.DoClick(h.addOk)

				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-2")
				h.dom.SetValue(h.addKey, "private-key-2")
				h.dom.DoClick(h.addOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key-1",
				},
				&displayedKey{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "add key cancelled by user",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, "private-key")
				h.dom.DoClick(h.addCancel)
			},
		},
		{
			description: "add key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "")
				h.dom.SetValue(h.addKey, "private-key")
				h.dom.DoClick(h.addOk)
			},
			wantErr: "failed to add key: invalid name: name must not be empty",
		},
		{
			description: "remove key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-1")
				h.dom.SetValue(h.addKey, "private-key-1")
				h.dom.DoClick(h.addOk)

				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-2")
				h.dom.SetValue(h.addKey, "private-key-2")
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.removeYes)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "remove key cancelled by user",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-1")
				h.dom.SetValue(h.addKey, "private-key-1")
				h.dom.DoClick(h.addOk)

				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-2")
				h.dom.SetValue(h.addKey, "private-key-2")
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.removeNo)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key-1",
				},
				&displayedKey{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "remove key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-1")
				h.dom.SetValue(h.addKey, "private-key-1")
				h.dom.DoClick(h.addOk)

				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key-2")
				h.dom.SetValue(h.addKey, "private-key-2")
				h.dom.DoClick(h.addOk)

				h.UI.remove(keys.ID("bogus-id"))
				h.dom.DoClick(h.removeYes)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key-1",
				},
				&displayedKey{
					ID:   validID,
					Name: "new-key-2",
				},
			},
			wantErr: "failed to remove key ID bogus-id: not found",
		},
		{
			description: "load key with passphrase",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				h.dom.DoClick(h.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.WithPassphrase.Type,
					Blob:   testdata.WithPassphrase.Blob,
				},
			},
		},
		{
			description: "load key cancelled by user",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.DoClick(h.passphraseCancel)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:        validID,
					Name:      "new-key",
					Encrypted: true,
				},
			},
		},
		{
			description: "load key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, "incorrect-passphrase")
				h.dom.DoClick(h.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:        validID,
					Name:      "new-key",
					Encrypted: true,
				},
			},
			wantErr: "failed to load key: failed to decrypt key: failed to parse private key: x509: decryption password incorrect",
		},
		{
			description: "load unencrypted key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithoutPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.WithoutPassphrase.Type,
					Blob:   testdata.WithoutPassphrase.Blob,
				},
			},
		},
		{
			description: "unload key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				h.dom.DoClick(h.passphraseOk)

				h.dom.DoClick(h.dom.GetElement(buttonID(UnloadButton, id)))
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:        validID,
					Name:      "new-key",
					Loaded:    false,
					Encrypted: true,
				},
			},
		},
		{
			description: "unload key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				h.dom.DoClick(h.passphraseOk)

				h.UI.unload(keys.ID("bogus-id"))
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.WithPassphrase.Type,
					Blob:   testdata.WithPassphrase.Blob,
				},
			},
			wantErr: "failed to unload key ID bogus-id: not found",
		},
		{
			description: "display non-configured keys",
			sequence: func(h *testHarness) {
				// Load an additional key directly into the agent.
				directLoadKey(h.agent, testdata.WithoutPassphrase.Private)

				// Configure a key of our own.
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				// Load the key we configured.
				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				h.dom.DoClick(h.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     keys.InvalidID,
					Loaded: true,
					Type:   testdata.WithoutPassphrase.Type,
					Blob:   testdata.WithoutPassphrase.Blob,
				},
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.WithPassphrase.Type,
					Blob:   testdata.WithPassphrase.Blob,
				},
			},
		},
		{
			description: "display loaded key that was previously-configured, then removed",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.addButton)
				h.dom.SetValue(h.addName, "new-key")
				h.dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				h.dom.DoClick(h.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				h.dom.DoClick(h.passphraseOk)

				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.removeYes)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     keys.InvalidID,
					Loaded: true,
					Type:   testdata.WithPassphrase.Type,
					Blob:   testdata.WithPassphrase.Blob,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			h := newHarness()
			defer h.Release()

			tc.sequence(h)

			displayed := equalizeIds(h.UI.displayedKeys())
			if diff := cmp.Diff(displayed, tc.wantDisplayed, displayedKeyCmp); diff != "" {
				t.Errorf("%s: incorrect displayed keys; -got +want: %s", tc.description, diff)
			}
			err := h.dom.TextContent(h.UI.errorText)
			if diff := cmp.Diff(err, tc.wantErr); diff != "" {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
		})
	}
}
