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

// Package optionsui defines the behavior underlying the user interface
// for the extension's options.
package optionsui

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"sort"
	"syscall/js"
	"time"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	"github.com/google/go-cmp/cmp"
)

// UI implements the behavior underlying the user interface for the extension's
// options.
type UI struct {
	mgr       keys.Manager
	dom       *dom.DOM
	addButton js.Value
	errorText js.Value
	keysData  js.Value
	keys      []*displayedKey
	cleanup   *jsutil.CleanupFuncs
}

type removeDialog struct {
	*dom.Dialog

	name js.Value
	yes  js.Value
	no   js.Value
}

type passphraseDialog struct {
	*dom.Dialog

	passphrase js.Value
	ok         js.Value
	cancel     js.Value
}

// New returns a new UI instance that manages keys using the supplied manager.
// domObj is the DOM instance corresponding to the document in which the Options
// UI is displayed.
func New(mgr keys.Manager, domObj *dom.DOM) *UI {
	result := &UI{
		mgr:       mgr,
		dom:       domObj,
		addButton: domObj.GetElement("add"),
		errorText: domObj.GetElement("errorMessage"),
		keysData:  domObj.GetElement("keysData"),
		cleanup:   &jsutil.CleanupFuncs{},
	}

	// Add event handlers.
	cf := result.cleanup
	// Populate keys on initial display
	cf.Add(result.dom.OnDOMContentLoaded(result.updateKeys))
	// Configure new key on click
	cf.Add(result.dom.OnClick(result.addButton, result.add))
	return result
}

// Release cleans up any resources when UI is no longer used.
func (u *UI) Release() {
	u.setKeys(nil)
	u.cleanup.Do()
}

// setError updates the UI to display the supplied error. If the supplied error
// is nil, then any displayed error is cleared.
func (u *UI) setError(err error) {
	// Clear any existing error
	u.dom.RemoveChildren(u.errorText)

	if err != nil {
		dom.LogError("UI.setError(): %v", err)
		u.dom.AppendChild(u.errorText, u.dom.NewText(err.Error()), nil)
	}
}

// add configures a new key.  It displays a dialog prompting the user for a name
// and the corresponding private key.  If the user continues, the key is
// added to the manager.
func (u *UI) add(evt dom.Event) {
	evt.PreventDefault()
	u.promptAdd(func(name, privateKey string) {
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
// callback is invoked when user clicks OK.
func (u *UI) promptAdd(onOk func(name, privateKey string)) {
	dialog := dom.NewDialog(u.dom.GetElement("addDialog"))
	name := u.dom.GetElement("addName")
	key := u.dom.GetElement("addKey")
	ok := u.dom.GetElement("addOk")
	cancel := u.dom.GetElement("addCancel")

	var cleanup jsutil.CleanupFuncs
	cleanup.Add(u.dom.OnClick(ok, func(evt dom.Event) {
		evt.PreventDefault()
		onOk(u.dom.Value(name), u.dom.Value(key))
		dialog.Close()
	}))
	cleanup.Add(u.dom.OnClick(cancel, func(evt dom.Event) {
		dialog.Close()
	}))
	cleanup.Add(dialog.OnClose(func(evt dom.Event) {
		u.dom.SetValue(name, "")
		u.dom.SetValue(key, "")
		cleanup.Do()
	}))

	dialog.ShowModal()
}

// load loads the key with the specified ID.  A dialog prompts the user for a
// passphrase if the private key is encrypted.
func (u *UI) load(id keys.ID) {
	k := u.keyByID(id)
	if k == nil {
		u.setError(fmt.Errorf("failed to unload key ID %s: not found", id))
		return
	}

	prompt := u.promptPassphrase
	if !k.Encrypted {
		// No passphrase required; use a dummy callback.
		prompt = func(onOk func(passphrase string)) { onOk("") }
	}

	prompt(func(passphrase string) {
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
// callback is invoked if user continues.
func (u *UI) promptPassphrase(onOk func(passphrase string)) {
	dialog := dom.NewDialog(u.dom.GetElement("passphraseDialog"))
	passphrase := u.dom.GetElement("passphrase")
	ok := u.dom.GetElement("passphraseOk")
	cancel := u.dom.GetElement("passphraseCancel")

	var cleanup jsutil.CleanupFuncs
	cleanup.Add(u.dom.OnClick(ok, func(evt dom.Event) {
		evt.PreventDefault()
		onOk(u.dom.Value(passphrase))
		dialog.Close()
	}))
	cleanup.Add(u.dom.OnClick(cancel, func(evt dom.Event) {
		dialog.Close()
	}))
	cleanup.Add(dialog.OnClose(func(evt dom.Event) {
		u.dom.SetValue(passphrase, "")
		cleanup.Do()
	}))

	dialog.ShowModal()
}

// unload unloads the specified key.
func (u *UI) unload(id keys.ID) {
	u.mgr.Unload(id, func(err error) {
		if err != nil {
			u.setError(fmt.Errorf("failed to unload key ID %s: %v", id, err))
			return
		}
		u.setError(nil)
		u.updateKeys()
	})
}

// promptRemove displays a dialog prompting the user to confirm that a key
// should be removed. callback is invoked if the key should be removed.
func (u *UI) promptRemove(id keys.ID, onYes func()) {
	k := u.keyByID(id)
	if k == nil {
		u.setError(fmt.Errorf("failed to remove key ID %s: not found", id))
		return
	}

	dialog := dom.NewDialog(u.dom.GetElement("removeDialog"))
	name := u.dom.GetElement("removeName")
	yes := u.dom.GetElement("removeYes")
	no := u.dom.GetElement("removeNo")
	u.dom.AppendChild(name, u.dom.NewText(k.Name), nil)

	var cleanup jsutil.CleanupFuncs
	cleanup.Add(u.dom.OnClick(yes, func(evt dom.Event) {
		evt.PreventDefault()
		onYes()
		dialog.Close()
	}))
	cleanup.Add(u.dom.OnClick(no, func(evt dom.Event) {
		dialog.Close()
	}))
	cleanup.Add(dialog.OnClose(func(evt dom.Event) {
		u.dom.RemoveChildren(name)
		cleanup.Do()
	}))

	dialog.ShowModal()
}

// remove removes the key with the specified ID.  A dialog prompts the user to
// confirm that the key should be removed.
func (u *UI) remove(id keys.ID) {
	u.promptRemove(id, func() {
		u.mgr.Remove(id, func(err error) {
			if err != nil {
				u.setError(fmt.Errorf("failed to remove key ID %s: %v", id, err))
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
	// Encrypted indicates if the private key is encrypted and requires a
	// passphrase to load. This field is only valid if the key is not
	// loaded.
	Encrypted bool
	// Name is the human-readable name assigned to the key.
	Name string
	// Type is the type of key (e.g., 'ssh-rsa').
	Type string
	// Blob is the public key material for the key.
	Blob string
	// Comment is the comment attached to the key in the agent
	Comment string
	// cleanup keeps track of any cleanup required before removing this key
	// from the UI.
	cleanup jsutil.CleanupFuncs
}

// LoadedKey returns the corresponding LoadedKey.
func (d *displayedKey) LoadedKey() (*keys.LoadedKey, error) {
	blob, err := base64.StdEncoding.DecodeString(d.Blob)
	if err != nil {
		return nil, fmt.Errorf("failed to decode blob: %v", err)
	}

	l := &keys.LoadedKey{
		Type:    d.Type,
		Comment: d.Comment,
	}
	l.SetBlob(blob)
	return l, nil
}

// displayedKeys returns the keys currently displayed in the UI.
func (u *UI) displayedKeys() []*displayedKey {
	return u.keys
}

func (u *UI) keyByID(id keys.ID) *displayedKey {
	if id == keys.InvalidID {
		return nil
	}

	for _, k := range u.keys {
		if k.ID == id {
			return k
		}
	}
	return nil
}

func (u *UI) keyByName(name string) *displayedKey {
	for _, k := range u.keys {
		if k.Name == name {
			return k
		}
	}
	return nil
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

// setKeys refreshes the UI to reflect the keys that should be
// displayed.
func (u *UI) setKeys(newKeys []*displayedKey) {
	// Cleanup elements and resources for all previous keys.
	u.dom.RemoveChildren(u.keysData)
	for _, k := range u.keys {
		k.cleanup.Do()
	}

	// Construct elements for new keys.
	u.keys = newKeys
	for _, k := range u.keys {
		k := k
		u.dom.AppendChild(u.keysData, u.dom.NewElement("tr"), func(row js.Value) {
			// Key name
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyName")
					u.dom.AppendChild(div, u.dom.NewText(k.Name), nil)
				})
			})

			// Controls
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyControls")
					if k.ID == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					if k.Loaded {
						// Unload button
						u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(UnloadButton, k.ID))
							u.dom.AppendChild(btn, u.dom.NewText("Unload"), nil)
							k.cleanup.Add(u.dom.OnClick(btn, func(evt dom.Event) {
								u.unload(k.ID)
							}))
						})
					} else {
						// Load button
						u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(LoadButton, k.ID))
							u.dom.AppendChild(btn, u.dom.NewText("Load"), nil)
							k.cleanup.Add(u.dom.OnClick(btn, func(evt dom.Event) {
								u.load(k.ID)
							}))
						})
					}

					// Remove button
					u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
						btn.Set("type", "button")
						btn.Set("id", buttonID(RemoveButton, k.ID))
						u.dom.AppendChild(btn, u.dom.NewText("Remove"), nil)
						k.cleanup.Add(u.dom.OnClick(btn, func(evt dom.Event) {
							u.remove(k.ID)
						}))
					})
				})
			})

			// Type
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyType")
					u.dom.AppendChild(div, u.dom.NewText(k.Type), nil)
				})
			})

			// Blob
			u.dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				u.dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
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
		configuredMap[keys.ID(k.ID)] = k
	}

	var result []*displayedKey

	// Add all loaded keys. Keep track of the IDs that were detected as
	// being loaded.
	loadedIds := make(map[keys.ID]bool)
	for _, l := range loaded {
		// Gather basic fields we get for any loaded key.
		dk := &displayedKey{
			Loaded:  true,
			Type:    l.Type,
			Blob:    base64.StdEncoding.EncodeToString(l.Blob()),
			Comment: l.Comment,
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
		if loadedIds[keys.ID(a.ID)] {
			continue
		}

		result = append(result, &displayedKey{
			ID:        keys.ID(a.ID),
			Loaded:    false,
			Encrypted: a.Encrypted,
			Name:      a.Name,
		})
	}

	// Sort to ensure consistent ordering.
	sort.Slice(result, func(i, j int) bool {
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
			u.setKeys(mergeKeys(configured, loaded))
		})
	})
}

const (
	pollInterval = 100 * time.Millisecond
	pollTimeout  = 5 * time.Second
)

func poll(done func() bool) {
	timeout := time.Now().Add(pollTimeout)
	for time.Now().Before(timeout) {
		if done() {
			return
		}
		time.Sleep(pollInterval)
	}
}

// EndToEndTest runs a set of tests via the UI.  Failures are returned as a list
// of errors.
//
// No attempt is made to clean up from any intermediate state should the test
// fail.
func (u *UI) EndToEndTest() []error {
	addButton := u.dom.GetElement("add")
	addName := u.dom.GetElement("addName")
	addKey := u.dom.GetElement("addKey")
	addOk := u.dom.GetElement("addOk")
	passphraseInput := u.dom.GetElement("passphrase")
	passphraseOk := u.dom.GetElement("passphraseOk")
	removeYes := u.dom.GetElement("removeYes")

	var errs []error

	dom.Log("Generate random name to use for key")
	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to generate random number: %v", err))
		return errs // Remaining tests have hard dependency on key name.
	}
	keyName := fmt.Sprintf("e2e-test-key-%s", i.String())

	dom.Log("Configure a new key")
	u.dom.DoClick(addButton)
	u.dom.SetValue(addName, keyName)
	// Use the long key to exercise storage of large values in Chrome storage.
	u.dom.SetValue(addKey, testdata.LongKeyWithPassphrase.Private)
	u.dom.DoClick(addOk)

	dom.Log("Validate configured keys; ensure new key is present")
	var key *displayedKey
	poll(func() bool {
		key = u.keyByName(keyName)
		return key != nil
	})
	if key == nil {
		errs = append(errs, fmt.Errorf("after added: failed to find key"))
		return errs // Remaining tests have hard dependency on configured key.
	}

	dom.Log("Load the new key")
	u.dom.DoClick(u.dom.GetElement(buttonID(LoadButton, key.ID)))
	u.dom.SetValue(passphraseInput, testdata.LongKeyWithPassphrase.Passphrase)
	u.dom.DoClick(passphraseOk)

	dom.Log("Validate loaded keys; ensure new key is loaded")
	poll(func() bool {
		key = u.keyByName(keyName)
		return key != nil && key.Loaded
	})
	if key != nil {
		if diff := cmp.Diff(key.Loaded, true); diff != "" {
			errs = append(errs, fmt.Errorf("after load: incorrect loaded state: %s", diff))
		}
		if diff := cmp.Diff(key.Type, testdata.LongKeyWithPassphrase.Type); diff != "" {
			errs = append(errs, fmt.Errorf("after load: incorrect type: %s", diff))
		}
		if diff := cmp.Diff(key.Blob, testdata.LongKeyWithPassphrase.Blob); diff != "" {
			errs = append(errs, fmt.Errorf("after load: incorrect blob: %s", diff))
		}
	} else if key == nil {
		errs = append(errs, fmt.Errorf("after load: failed to find key"))
	}

	dom.Log("Unload key")
	u.dom.DoClick(u.dom.GetElement(buttonID(UnloadButton, key.ID)))

	dom.Log("Validate loaded keys; ensure key is unloaded")
	poll(func() bool {
		key = u.keyByName(keyName)
		return key != nil && !key.Loaded
	})
	if key != nil {
		if diff := cmp.Diff(key.Loaded, false); diff != "" {
			errs = append(errs, fmt.Errorf("after unload: incorrect loaded state: %s", diff))
		}
		if diff := cmp.Diff(key.Type, ""); diff != "" {
			errs = append(errs, fmt.Errorf("after unload: incorrect type: %s", diff))
		}
		if diff := cmp.Diff(key.Blob, ""); diff != "" {
			errs = append(errs, fmt.Errorf("after unload: incorrect blob: %s", diff))
		}
	} else if key == nil {
		errs = append(errs, fmt.Errorf("after unload: failed to find key"))
	}

	dom.Log("Remove key")
	u.dom.DoClick(u.dom.GetElement(buttonID(RemoveButton, key.ID)))
	u.dom.DoClick(removeYes)

	dom.Log("Validate configured keys; ensure key is removed")
	poll(func() bool {
		key = u.keyByName(keyName)
		return key == nil
	})
	if key != nil {
		errs = append(errs, fmt.Errorf("after removed: incorrectly found key"))
	}

	return errs

}
