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
	"io/ioutil"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/google/chrome-ssh-agent/go/dom"
	dt "github.com/google/chrome-ssh-agent/go/dom/testing"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	"github.com/kr/pretty"
)

var (
	validID = keys.ID("1")

	optionsHTML = ""
)

func init() {
	b, err := ioutil.ReadFile("../../html/options.html")
	if err != nil {
		panic(fmt.Sprintf("failed to read options html: %v", err))
	}

	optionsHTML = string(b)
}

type testHarness struct {
	storage   *fakes.MemStorage
	messaging *fakes.MessageHub
	agent     agent.Agent
	manager   keys.Manager
	server    *keys.Server
	Client    keys.Manager
	dom       *dom.DOM
	UI        *UI
}

func newHarness() *testHarness {
	storage := fakes.NewMemStorage()
	msg := fakes.NewMessageHub()

	agt := agent.NewKeyring()
	mgr := keys.NewManager(agt, storage)
	srv := keys.NewServer(mgr, msg)
	cli := keys.NewClient(msg)
	dom := dom.New(dt.NewDocForTesting(optionsHTML))
	ui := New(cli, dom)

	// In our test, DOMContentLoaded is not called automatically. Do it here.
	dom.DoDOMContentLoaded()

	return &testHarness{
		storage:   storage,
		messaging: msg,
		agent:     agt,
		manager:   mgr,
		server:    srv,
		Client:    cli,
		dom:       dom,
		UI:        ui,
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
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addOk)
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
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-1")
				h.dom.SetValue(h.UI.addKey, "private-key-1")
				h.dom.DoClick(h.UI.addOk)

				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-2")
				h.dom.SetValue(h.UI.addKey, "private-key-2")
				h.dom.DoClick(h.UI.addOk)
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
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addCancel)
			},
		},
		{
			description: "add key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addOk)
			},
			wantErr: "failed to add key: name must not be empty",
		},
		{
			description: "remove key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-1")
				h.dom.SetValue(h.UI.addKey, "private-key-1")
				h.dom.DoClick(h.UI.addOk)

				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-2")
				h.dom.SetValue(h.UI.addKey, "private-key-2")
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.UI.removeYes)
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
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-1")
				h.dom.SetValue(h.UI.addKey, "private-key-1")
				h.dom.DoClick(h.UI.addOk)

				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-2")
				h.dom.SetValue(h.UI.addKey, "private-key-2")
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key-1")
				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.UI.removeNo)
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
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-1")
				h.dom.SetValue(h.UI.addKey, "private-key-1")
				h.dom.DoClick(h.UI.addOk)

				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key-2")
				h.dom.SetValue(h.UI.addKey, "private-key-2")
				h.dom.DoClick(h.UI.addOk)

				h.UI.remove(keys.ID("bogus-id"), "bogus-name")
				h.dom.DoClick(h.UI.removeYes)
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
			// It would be nice to return an error here, but
			// keys.Manager.Remove does not.  See keys.Manager.Remove
			// for details.
		},
		{
			description: "load key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.UI.passphraseInput, testdata.ValidPrivateKeyPassphrase)
				h.dom.DoClick(h.UI.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.ValidPrivateKeyType,
					Blob:   testdata.ValidPrivateKeyBlob,
				},
			},
		},
		{
			description: "load key",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.UI.passphraseInput, testdata.ValidPrivateKeyPassphrase)
				h.dom.DoClick(h.UI.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.ValidPrivateKeyType,
					Blob:   testdata.ValidPrivateKeyBlob,
				},
			},
		},
		{
			description: "load key cancelled by user",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.DoClick(h.UI.passphraseCancel)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key",
				},
			},
		},
		{
			description: "load key fails",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.UI.passphraseInput, "incorrect-passphrase")
				h.dom.DoClick(h.UI.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:   validID,
					Name: "new-key",
				},
			},
			wantErr: "failed to load key: failed to parse private key: x509: decryption password incorrect",
		},
		{
			description: "display non-configured keys",
			sequence: func(h *testHarness) {
				// Load an additional key directly into the agent.
				directLoadKey(h.agent, testdata.ValidPrivateKeyWithoutPassphrase)

				// Configure a key of our own.
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				// Load the key we configured.
				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.UI.passphraseInput, testdata.ValidPrivateKeyPassphrase)
				h.dom.DoClick(h.UI.passphraseOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     keys.InvalidID,
					Loaded: true,
					Type:   testdata.ValidPrivateKeyWithoutPassphraseType,
					Blob:   testdata.ValidPrivateKeyWithoutPassphraseBlob,
				},
				&displayedKey{
					ID:     validID,
					Name:   "new-key",
					Loaded: true,
					Type:   testdata.ValidPrivateKeyType,
					Blob:   testdata.ValidPrivateKeyBlob,
				},
			},
		},
		{
			description: "display loaded key that was previously-configured, then removed",
			sequence: func(h *testHarness) {
				h.dom.DoClick(h.UI.addButton)
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, testdata.ValidPrivateKey)
				h.dom.DoClick(h.UI.addOk)

				id := findKey(h.UI.displayedKeys(), "new-key")
				h.dom.DoClick(h.dom.GetElement(buttonID(LoadButton, id)))
				h.dom.SetValue(h.UI.passphraseInput, testdata.ValidPrivateKeyPassphrase)
				h.dom.DoClick(h.UI.passphraseOk)

				h.dom.DoClick(h.dom.GetElement(buttonID(RemoveButton, id)))
				h.dom.DoClick(h.UI.removeYes)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					ID:     keys.InvalidID,
					Loaded: true,
					Type:   testdata.ValidPrivateKeyType,
					Blob:   testdata.ValidPrivateKeyBlob,
				},
			},
		},
	}

	for _, tc := range testcases {
		h := newHarness()
		tc.sequence(h)

		displayed := equalizeIds(h.UI.displayedKeys())
		if diff := pretty.Diff(displayed, tc.wantDisplayed); diff != nil {
			t.Errorf("%s: incorrect displayed keys; -got +want: %s", tc.description, diff)
		}
		err := h.dom.TextContent(h.UI.errorText)
		if diff := pretty.Diff(err, tc.wantErr); diff != nil {
			t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
		}
	}
}
