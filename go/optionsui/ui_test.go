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
	"testing"

	"golang.org/x/crypto/ssh/agent"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

var (
	dummys = js.Global.Call("eval", `({
		newDoc: function(html) {
			const jsdom = require("jsdom");
			const virtualConsole = new jsdom.VirtualConsole();
			virtualConsole.sendTo(console);
			const { JSDOM } = jsdom;
			const dom = new JSDOM(html);
			return dom.window.document;
		},
	})`)

	validId = keys.ID("1")

	// TODO(ralimi) Fill this in dynamically from options.html
	// instead of copying and pasting.
	html = `
<!DOCTYPE html>
<html>
  <head>
    <title>Chrome SSH Agent</title>
    <link rel="stylesheet" href="style.css"/>
  </head>

  <body class="body">
    <dialog id="passphraseDialog" class="dialog">
      <div class="modal-content">
        <div>
          <label for="passphrase">Passphrase</label>
        </div>
        <div>
          <input id="passphrase" name="passphrase" type="password"/>
        </div>
        <div>
          <button id="passphraseOk">OK</button>
          <button id="passphraseCancel">Cancel</button>
        </div>
      </div>
    </dialog>

    <dialog id="addDialog" class="dialog">
      <div class="dialog-content">
        <div>
          <label for="addName">Name</label>
        </div>
        <div>
          <input id="addName" name="name" type="text"/>
        </div>
        <div>
          <label for="addKey">Private Key (PEM format)</label>
        </div>
        <div>
          <textarea id="addKey" name="privateKey"></textarea>
        </div>
        <div>
          <button id="addOk">Add</button>
          <button id="addCancel">Cancel</button>
        </div>
      </div>
    </dialog>

    <div id="options">
      <div id="errorMessage"></div>

      <div id="controlPane">
        <button id="add">Add Key</button>
      </div>

      <div id="keysPane">
        <table id="keysTable">
          <thead id="keysHeader">
            <tr>
              <td>Name</td>
              <td>Controls</td>
              <td>Type</td>
              <td>Blob</td>
            </tr>
          </thead>
          <tbody id="keysData">
          </tbody>
        </table>
      </div>
    </div>

    <script src="../go/options/options.js"></script>
  </body>
</html>
	`
)

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
	dom := dom.New(dummys.Call("newDoc", html))
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
		UI:        ui,
	}
}

func equalizeIds(disp []*displayedKey) []*displayedKey {
	var result []*displayedKey
	for _, k := range disp {
		nk := *k
		if nk.Id != keys.InvalidID {
			nk.Id = validId
		}
		result = append(result, &nk)
	}
	return result
}

func TestAddKey(t *testing.T) {
	testcases := []struct {
		description   string
		sequence      func(h *testHarness)
		wantDisplayed []*displayedKey
		wantErr       string
	}{
		{
			description: "add key",
			sequence: func(h *testHarness) {
				h.UI.Add()
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addOk)
			},
			wantDisplayed: []*displayedKey{
				&displayedKey{
					Id:   validId,
					Name: "new-key",
				},
			},
		},
		{
			description: "add key cancelled by user",
			sequence: func(h *testHarness) {
				h.UI.Add()
				h.dom.SetValue(h.UI.addName, "new-key")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addCancel)
			},
		},
		{
			description: "add key fails",
			sequence: func(h *testHarness) {
				h.UI.Add()
				h.dom.SetValue(h.UI.addName, "")
				h.dom.SetValue(h.UI.addKey, "private-key")
				h.dom.DoClick(h.UI.addOk)
			},
			wantErr: "failed to add key: name must not be empty",
		},
	}

	for _, tc := range testcases {
		h := newHarness()
		tc.sequence(h)

		displayed := equalizeIds(h.UI.DisplayedKeys())
		if diff := pretty.Diff(displayed, tc.wantDisplayed); diff != nil {
			t.Errorf("%s: incorrect displayed keys; -got +want: %s", tc.description, diff)
		}
		err := h.dom.TextContent(h.UI.errorText)
		if diff := pretty.Diff(err, tc.wantErr); diff != nil {
			t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
		}
	}
}
