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
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/google/chrome-ssh-agent/go/dom"
	dt "github.com/google/chrome-ssh-agent/go/dom/testing"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	mfakes "github.com/google/chrome-ssh-agent/go/message/fakes"
	"github.com/google/chrome-ssh-agent/go/storage"
	st "github.com/google/chrome-ssh-agent/go/storage/testing"
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
	messaging *mfakes.Hub
	agent     agent.Agent
	manager   keys.Manager
	server    *keys.Server
	Client    keys.Manager
	dom       *dom.Doc
	UI        *UI

	loadingText      js.Value
	addDialog        js.Value
	addButton        js.Value
	addName          js.Value
	addKey           js.Value
	addOk            js.Value
	addCancel        js.Value
	passphraseDialog js.Value
	passphraseInput  js.Value
	passphraseOk     js.Value
	passphraseCancel js.Value
	removeDialog     js.Value
	removeYes        js.Value
	removeNo         js.Value
}

func (h *testHarness) Release() {
	h.UI.Release()
}

func mustPoll(ctx jsutil.AsyncContext, done func() bool) {
	if !poll(ctx, done) {
		panic("timed out waiting for condition")
	}
}

func (h *testHarness) waitLoaded(ctx jsutil.AsyncContext) {
	mustPoll(ctx, func() bool { return dom.TextContent(h.loadingText) == "" })
}

func (h *testHarness) waitDialogOpen(ctx jsutil.AsyncContext, dialog js.Value) {
	mustPoll(ctx, func() bool { return dialog.Get("open").Bool() })
}

func (h *testHarness) waitDialogClosed(ctx jsutil.AsyncContext, dialog js.Value) {
	mustPoll(ctx, func() bool { return !dialog.Get("open").Bool() })
}

func (h *testHarness) waitKeyConfigured(ctx jsutil.AsyncContext, name string) {
	mustPoll(ctx, func() bool { return h.UI.keyByName(name) != nil })
}

func (h *testHarness) waitKeyRemoved(ctx jsutil.AsyncContext, name string) {
	mustPoll(ctx, func() bool { return h.UI.keyByName(name) == nil })
}

func (h *testHarness) waitKeyLoaded(ctx jsutil.AsyncContext, name string) {
	mustPoll(ctx, func() bool {
		k := h.UI.keyByName(name)
		return k != nil && k.Loaded
	})
}

func (h *testHarness) waitKeyUnloaded(ctx jsutil.AsyncContext, name string) {
	mustPoll(ctx, func() bool {
		k := h.UI.keyByName(name)
		return k != nil && !k.Loaded
	})
}

func newHarness() *testHarness {
	syncStorage := storage.NewRaw(st.NewMemArea())
	sessionStorage := storage.NewRaw(st.NewMemArea())
	msg := mfakes.NewHub()

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
		loadingText:      domObj.GetElement("loadingMessage"),
		addDialog:        domObj.GetElement("addDialog"),
		addButton:        domObj.GetElement("add"),
		addName:          domObj.GetElement("addName"),
		addKey:           domObj.GetElement("addKey"),
		addOk:            domObj.GetElement("addOk"),
		addCancel:        domObj.GetElement("addCancel"),
		passphraseDialog: domObj.GetElement("passphraseDialog"),
		passphraseInput:  domObj.GetElement("passphrase"),
		passphraseOk:     domObj.GetElement("passphraseOk"),
		passphraseCancel: domObj.GetElement("passphraseCancel"),
		removeDialog:     domObj.GetElement("removeDialog"),
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
		sequence      func(ctx jsutil.AsyncContext, h *testHarness)
		wantDisplayed []*displayedKey
		wantErr       string
	}{
		{
			description: "add key",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, "private-key")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")
			},
			wantDisplayed: []*displayedKey{
				{
					ID:   validID,
					Name: "new-key",
				},
			},
		},
		{
			description: "add multiple keys",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-1")
				dom.SetValue(h.addKey, "private-key-1")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-1")

				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-2")
				dom.SetValue(h.addKey, "private-key-2")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-2")
			},
			wantDisplayed: []*displayedKey{
				{
					ID:   validID,
					Name: "new-key-1",
				},
				{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "add key cancelled by user",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, "private-key")
				dom.DoClick(h.addCancel)
				h.waitDialogClosed(ctx, h.addDialog)
			},
		},
		{
			description: "add key fails",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "")
				dom.SetValue(h.addKey, "private-key")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
			},
			wantErr: "failed to add key: invalid name: name must not be empty",
		},
		{
			description: "remove key",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-1")
				dom.SetValue(h.addKey, "private-key-1")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-1")

				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-2")
				dom.SetValue(h.addKey, "private-key-2")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-2")

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.waitDialogOpen(ctx, h.removeDialog)
				dom.DoClick(h.removeYes)
				h.waitDialogClosed(ctx, h.removeDialog)
				h.waitKeyRemoved(ctx, "new-key-1")
			},
			wantDisplayed: []*displayedKey{
				{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "remove key cancelled by user",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-1")
				dom.SetValue(h.addKey, "private-key-1")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-1")

				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-2")
				dom.SetValue(h.addKey, "private-key-2")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-2")

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.waitDialogOpen(ctx, h.removeDialog)
				dom.DoClick(h.removeNo)
				h.waitDialogClosed(ctx, h.removeDialog)
			},
			wantDisplayed: []*displayedKey{
				{
					ID:   validID,
					Name: "new-key-1",
				},
				{
					ID:   validID,
					Name: "new-key-2",
				},
			},
		},
		{
			description: "remove key fails",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-1")
				dom.SetValue(h.addKey, "private-key-1")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-1")

				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key-2")
				dom.SetValue(h.addKey, "private-key-2")
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key-2")

				h.UI.remove(ctx, keys.ID("bogus-id"))
				dom.DoClick(h.removeYes)
				h.waitDialogClosed(ctx, h.removeDialog)
			},
			wantDisplayed: []*displayedKey{
				{
					ID:   validID,
					Name: "new-key-1",
				},
				{
					ID:   validID,
					Name: "new-key-2",
				},
			},
			wantErr: "failed to remove key ID bogus-id: not found",
		},
		{
			description: "load key with passphrase",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
				h.waitKeyLoaded(ctx, "new-key")
			},
			wantDisplayed: []*displayedKey{
				{
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
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.DoClick(h.passphraseCancel)
				h.waitDialogClosed(ctx, h.passphraseDialog)
			},
			wantDisplayed: []*displayedKey{
				{
					ID:        validID,
					Name:      "new-key",
					Encrypted: true,
				},
			},
		},
		{
			description: "load key fails",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, "incorrect-passphrase")
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
			},
			wantDisplayed: []*displayedKey{
				{
					ID:        validID,
					Name:      "new-key",
					Encrypted: true,
				},
			},
			wantErr: "failed to load key: failed to decrypt key: failed to parse private key: x509: decryption password incorrect",
		},
		{
			description: "load unencrypted key",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithoutPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
			},
			wantDisplayed: []*displayedKey{
				{
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
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
				h.waitKeyLoaded(ctx, "new-key")

				dom.DoClick(h.dom.GetElement(buttonID(UnloadButton, id)))
				h.waitKeyUnloaded(ctx, "new-key")
			},
			wantDisplayed: []*displayedKey{
				{
					ID:        validID,
					Name:      "new-key",
					Loaded:    false,
					Encrypted: true,
				},
			},
		},
		{
			description: "unload key fails",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
				h.waitKeyLoaded(ctx, "new-key")

				h.UI.unload(ctx, keys.ID("bogus-id"))
			},
			wantDisplayed: []*displayedKey{
				{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.WithPassphrase.Type,
					Blob:   testdata.WithPassphrase.Blob,
				},
			},
			wantErr: "failed to unload key ID bogus-id: key unload from agent failed: invalid id: bogus-id",
		},
		{
			description: "display non-configured keys",
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				// Load an additional key directly into the agent.
				directLoadKey(h.agent, testdata.WithoutPassphrase.Private)

				// Configure a key of our own.
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				// Load the key we configured.
				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
				h.waitKeyLoaded(ctx, "new-key")
			},
			wantDisplayed: []*displayedKey{
				{
					ID:     keys.InvalidID,
					Loaded: true,
					Type:   testdata.WithoutPassphrase.Type,
					Blob:   testdata.WithoutPassphrase.Blob,
				},
				{
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
			sequence: func(ctx jsutil.AsyncContext, h *testHarness) {
				dom.DoClick(h.addButton)
				h.waitDialogOpen(ctx, h.addDialog)
				dom.SetValue(h.addName, "new-key")
				dom.SetValue(h.addKey, testdata.WithPassphrase.Private)
				dom.DoClick(h.addOk)
				h.waitDialogClosed(ctx, h.addDialog)
				h.waitKeyConfigured(ctx, "new-key")

				id := findKey(h.UI.displayedKeys(), "new-key")
				dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.waitDialogOpen(ctx, h.passphraseDialog)
				dom.SetValue(h.passphraseInput, testdata.WithPassphrase.Passphrase)
				dom.DoClick(h.passphraseOk)
				h.waitDialogClosed(ctx, h.passphraseDialog)
				h.waitKeyLoaded(ctx, "new-key")

				dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.waitDialogOpen(ctx, h.removeDialog)
				dom.DoClick(h.removeYes)
				h.waitDialogClosed(ctx, h.removeDialog)
				h.waitKeyRemoved(ctx, "new-key")
			},
			wantDisplayed: []*displayedKey{
				{
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

			jut.DoSync(func(ctx jsutil.AsyncContext) {
				h.waitLoaded(ctx)
				tc.sequence(ctx, h)
				// Give some buffer for any pending async
				// operations to settle.
				time.Sleep(50 * time.Millisecond)
			})

			displayed := equalizeIds(h.UI.displayedKeys())
			if diff := cmp.Diff(displayed, tc.wantDisplayed, displayedKeyCmp); diff != "" {
				t.Errorf("%s: incorrect displayed keys; -got +want: %s", tc.description, diff)
			}
			err := dom.TextContent(h.UI.errorText)
			if diff := cmp.Diff(err, tc.wantErr); diff != "" {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
		})
	}
}
