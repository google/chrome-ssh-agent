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

// Package optionsui defines the behavior underlying the user interface
// for the extension's options.
package optionsui

import (
	"encoding/base64"
	"fmt"
	"sort"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/gopherjs/gopherjs/js"
)

// UI implements the behavior underlying the user interface for the extension's
// options.
type UI struct {
	mgr              keys.Manager
	dom              *dom.DOM
	passphraseDialog *js.Object
	passphraseInput  *js.Object
	passphraseOk     *js.Object
	passphraseCancel *js.Object
	addButton        *js.Object
	addDialog        *js.Object
	addName          *js.Object
	addKey           *js.Object
	addOk            *js.Object
	addCancel        *js.Object
	removeDialog     *js.Object
	removeName       *js.Object
	removeYes        *js.Object
	removeNo         *js.Object
	errorText        *js.Object
	keysData         *js.Object
	keys             []*displayedKey
}

// New returns a new UI instance that manages keys using the supplied manager.
// domObj is the DOM instance corresponding to the document in which the Options
// UI is displayed.
func New(mgr keys.Manager, domObj *dom.DOM) *UI {
	result := &UI{
		mgr:              mgr,
		dom:              domObj,
		passphraseDialog: domObj.GetElement("passphraseDialog"),
		passphraseInput:  domObj.GetElement("passphrase"),
		passphraseOk:     domObj.GetElement("passphraseOk"),
		passphraseCancel: domObj.GetElement("passphraseCancel"),
		addButton:        domObj.GetElement("add"),
		addDialog:        domObj.GetElement("addDialog"),
		addName:          domObj.GetElement("addName"),
		addKey:           domObj.GetElement("addKey"),
		addOk:            domObj.GetElement("addOk"),
		addCancel:        domObj.GetElement("addCancel"),
		removeDialog:     domObj.GetElement("removeDialog"),
		removeName:       domObj.GetElement("removeName"),
		removeYes:        domObj.GetElement("removeYes"),
		removeNo:         domObj.GetElement("removeNo"),
		errorText:        domObj.GetElement("errorMessage"),
		keysData:         domObj.GetElement("keysData"),
	}

	// Populate keys on initial display
	result.dom.OnDOMContentLoaded(result.updateKeys)
	// Configure new key on click
	result.dom.OnClick(result.addButton, result.add)
	return result
}

// setError updates the UI to display the supplied error. If the supplied error
// is nil, then any displayed error is cleared.
func (u *UI) setError(err error) {
	// Clear any existing error
	u.dom.RemoveChildren(u.errorText)

	if err != nil {
		u.dom.AppendChild(u.errorText, u.dom.NewText(err.Error()), nil)
	}
}

// add configures a new key.  It displays a dialog prompting the user for a name
// and the corresponding private key.  If the user continues, the key is
// added to the manager.
func (u *UI) add() {
	u.promptAdd(func(name, privateKey string, ok bool) {
		if !ok {
			return
		}
		u.mgr.Add(name, privateKey, func(err error) {
			if err != nil {
				u.setError(fmt.Errorf("failed to add key: %v", err))
				return
			}

			u.setError(nil)
			u.updateKeys()
		})
	})
}

// promptAdd displays a dialog prompting the user for a name and private key.
// callback is invoked when the dialog is closed; the ok parameter indicates
// if the user clicked OK.
func (u *UI) promptAdd(callback func(name, privateKey string, ok bool)) {
	u.dom.OnClick(u.addOk, func() {
		n := u.dom.Value(u.addName)
		k := u.dom.Value(u.addKey)
		u.dom.SetValue(u.addName, "")
		u.dom.SetValue(u.addKey, "")
		u.addOk = u.dom.RemoveEventListeners(u.addOk)
		u.addCancel = u.dom.RemoveEventListeners(u.addCancel)
		u.dom.Close(u.addDialog)
		callback(n, k, true)
	})
	u.dom.OnClick(u.addCancel, func() {
		u.dom.SetValue(u.addName, "")
		u.dom.SetValue(u.addKey, "")
		u.addOk = u.dom.RemoveEventListeners(u.addOk)
		u.addCancel = u.dom.RemoveEventListeners(u.addCancel)
		u.dom.Close(u.addDialog)
		callback("", "", false)
	})
	u.dom.ShowModal(u.addDialog)
}

// load loads the key with the specified ID.  A dialog prompts the user for a
// passphrase.
func (u *UI) load(id keys.ID) {
	u.promptPassphrase(func(passphrase string, ok bool) {
		if !ok {
			return
		}
		u.mgr.Load(id, passphrase, func(err error) {
			if err != nil {
				u.setError(fmt.Errorf("failed to load key: %v", err))
				return
			}
			u.setError(nil)
			u.updateKeys()
		})
	})
}

// promptPassphrase displays a dialog prompting the user for a passphrase.
// callback is invoked when the dialog is closed; the ok parameter indicates
// if the user clicked OK.
func (u *UI) promptPassphrase(callback func(passphrase string, ok bool)) {
	u.dom.OnClick(u.passphraseOk, func() {
		p := u.dom.Value(u.passphraseInput)
		u.dom.SetValue(u.passphraseInput, "")
		u.passphraseOk = u.dom.RemoveEventListeners(u.passphraseOk)
		u.passphraseCancel = u.dom.RemoveEventListeners(u.passphraseCancel)
		u.dom.Close(u.passphraseDialog)
		callback(p, true)
	})
	u.dom.OnClick(u.passphraseCancel, func() {
		u.dom.SetValue(u.passphraseInput, "")
		u.passphraseOk = u.dom.RemoveEventListeners(u.passphraseOk)
		u.passphraseCancel = u.dom.RemoveEventListeners(u.passphraseCancel)
		u.dom.Close(u.passphraseDialog)
		callback("", false)
	})
	u.dom.ShowModal(u.passphraseDialog)
}

// unload unloads the specified key.
func (u *UI) unload(key *keys.LoadedKey) {
	u.mgr.Unload(key, func(err error) {
		if err != nil {
			u.setError(fmt.Errorf("failed to unload key: %v", err))
			return
		}
		u.setError(nil)
		u.updateKeys()
	})
}

// promptRemove displays a dialog prompting the user to confirm that a key
// should be removed. callback is invoked when the dialog is closed; the yes
// parameter indicates if the user clicked Yes.
func (u *UI) promptRemove(name string, callback func(yes bool)) {
	u.dom.RemoveChildren(u.removeName)
	u.dom.AppendChild(u.removeName, u.dom.NewText(name), nil)
	u.dom.OnClick(u.removeYes, func() {
		u.dom.RemoveChildren(u.removeName)
		u.removeYes = u.dom.RemoveEventListeners(u.removeYes)
		u.removeNo = u.dom.RemoveEventListeners(u.removeNo)
		u.dom.Close(u.removeDialog)
		callback(true)
	})
	u.dom.OnClick(u.removeNo, func() {
		u.dom.RemoveChildren(u.removeName)
		u.removeYes = u.dom.RemoveEventListeners(u.removeYes)
		u.removeNo = u.dom.RemoveEventListeners(u.removeNo)
		u.dom.Close(u.removeDialog)
		callback(false)
	})
	u.dom.ShowModal(u.removeDialog)
}

// remove removes the key with the specified ID.  A dialog prompts the user to
// confirm that the key should be removed.
func (u *UI) remove(id keys.ID, name string) {
	u.promptRemove(name, func(yes bool) {
		if !yes {
			return
		}

		u.mgr.Remove(id, func(err error) {
			if err != nil {
				u.setError(fmt.Errorf("failed to remove key: %v", err))
				return
			}

			u.setError(nil)
			u.updateKeys()
		})
	})
}

// displayedKey represents a key displayed in the UI.
type displayedKey struct {
	// ID is the unique ID corresponding to the key.
	ID keys.ID
	// Loaded indicates if the key is currently loaded.
	Loaded bool
	// Name is the human-readable name assigned to the key.
	Name string
	// Type is the type of key (e.g., 'ssh-rsa').
	Type string
	// Blob is the public key material for the key.
	Blob string
}

func (d *displayedKey) LoadedKey() (*keys.LoadedKey, error) {
	blob, err := base64.StdEncoding.DecodeString(d.Blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode blob: %v", err)
	}

	l := &keys.LoadedKey{Object: js.Global.Get("Object").New()}
	l.Type = d.Type
	l.SetBlob(blob)
	return l, nil
}

// DisplayedKeys returns the keys currently displayed in the UI.
func (u *UI) displayedKeys() []*displayedKey {
	return u.keys
}

// buttonKind is the type of button displayed for a key.
type buttonKind int

const (
	// LoadButton indicates that the button loads the key into the agent.
	LoadButton buttonKind = iota
	// UnloadButton indicates that the button unloads the key from the
	// agent.
	UnloadButton
	// RemoveButton indicates that the button removes the key.
	RemoveButton
)

// buttonID returns the value of the 'id' attribute to be assigned to the HTML
// button.
func buttonID(kind buttonKind, id keys.ID) string {
	s := "unknown"
	switch kind {
	case LoadButton:
		s = "load"
	case UnloadButton:
		s = "unload"
	case RemoveButton:
		s = "remove"
	}
	return fmt.Sprintf("%s-%s", s, id)
}

// updateDisplayedKeys refreshes the UI to reflect the keys that should be
// displayed.
func (u *UI) updateDisplayedKeys() {
	u.dom.RemoveChildren(u.keysData)

	for _, k := range u.keys {
		k := k
		u.dom.AppendChild(u.keysData, u.dom.NewElement("tr"), func(row *js.Object) {
			// Key name
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell *js.Object) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyName")
					u.dom.AppendChild(div, u.dom.NewText(k.Name), nil)
				})
			})

			// Controls
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell *js.Object) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyControls")
					if k.ID == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					if k.Loaded {
						// Unload button
						u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn *js.Object) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(UnloadButton, k.ID))
							u.dom.AppendChild(btn, u.dom.NewText("Unload"), nil)
							u.dom.OnClick(btn, func() {
								l, err := k.LoadedKey()
								if err != nil {
									u.setError(fmt.Errorf("Failed to get loaded key: %v", err))
									return
								}
								u.unload(l)
							})
						})
					} else {
						// Load button
						u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn *js.Object) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(LoadButton, k.ID))
							u.dom.AppendChild(btn, u.dom.NewText("Load"), nil)
							u.dom.OnClick(btn, func() {
								u.load(k.ID)
							})
						})
					}

					// Remove button
					u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn *js.Object) {
						btn.Set("type", "button")
						btn.Set("id", buttonID(RemoveButton, k.ID))
						u.dom.AppendChild(btn, u.dom.NewText("Remove"), nil)
						u.dom.OnClick(btn, func() {
							u.remove(k.ID, k.Name)
						})
					})
				})
			})

			// Type
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell *js.Object) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyType")
					u.dom.AppendChild(div, u.dom.NewText(k.Type), nil)
				})
			})

			// Blob
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell *js.Object) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyBlob")
					u.dom.AppendChild(div, u.dom.NewText(k.Blob), nil)
				})
			})
		})
	}
}

// mergeKeys merges configured and loaded keys to create a consolidated list
// of keys that should be displayed in the UI.
func mergeKeys(configured []*keys.ConfiguredKey, loaded []*keys.LoadedKey) []*displayedKey {
	// Build map of configured keys for faster lookup
	configuredMap := make(map[keys.ID]*keys.ConfiguredKey)
	for _, k := range configured {
		configuredMap[k.ID] = k
	}

	var result []*displayedKey

	// Add all loaded keys. Keep track of the IDs that were detected as
	// being loaded.
	loadedIds := make(map[keys.ID]bool)
	for _, l := range loaded {
		// Gather basic fields we get for any loaded key.
		dk := &displayedKey{
			Loaded: true,
			Type:   l.Type,
			Blob:   base64.StdEncoding.EncodeToString(l.Blob()),
		}
		// Attempt to figure out if this is a key we loaded. If so, fill
		// in some additional information.  It is possible that a key with
		// a non-existent ID is loaded (e.g., it was removed while loaded);
		// in this case we claim we do not have an ID.
		if id := l.ID(); id != keys.InvalidID {
			if ak := configuredMap[id]; ak != nil {
				loadedIds[id] = true
				dk.ID = id
				dk.Name = ak.Name
			}
		}
		result = append(result, dk)
	}

	// Add all configured keys that are not loaded.
	for _, a := range configured {
		// Skip any that we already covered above.
		if loadedIds[a.ID] {
			continue
		}

		result = append(result, &displayedKey{
			ID:     a.ID,
			Loaded: false,
			Name:   a.Name,
		})
	}

	// Sort to ensure consistent ordering.
	sort.Slice(result, func (i, j int) bool {
		a, b := result[i], result[j]
		if a.Name < b.Name {
			return true
		}
		if a.Name > b.Name {
			return false
		}
		if a.Blob < b.Blob {
			return true
		}
		if a.Blob > b.Blob {
			return false
		}
		return a.ID < b.ID
	})

	return result
}

// updateKeys queries the manager for configured and loaded keys, then triggers
// UI updates to reflect the current state.
func (u *UI) updateKeys() {
	u.mgr.Configured(func(configured []*keys.ConfiguredKey, err error) {
		if err != nil {
			u.setError(fmt.Errorf("failed to get configured keys: %v", err))
			return
		}

		u.mgr.Loaded(func(loaded []*keys.LoadedKey, err error) {
			if err != nil {
				u.setError(fmt.Errorf("failed to get loaded keys: %v", err))
				return
			}

			u.setError(nil)
			u.keys = mergeKeys(configured, loaded)
			u.updateDisplayedKeys()
		})
	})
}
