//go:build js

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
	"sync"
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
	dom       *dom.Doc
	addButton js.Value
	errorText js.Value
	keysData  js.Value
	keys      []*displayedKey
	cleanup   *jsutil.CleanupFuncs
}

// signal is a primitive that allows one routine to block until notified.
//
// It is a simple wrapper around WaitGroup that ensures blocking is invoked
// within an AsyncContext.
type signal struct {
	wg *sync.WaitGroup
}

// newSignal returns a new signal in the unnotified state.
func newSignal() *signal {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	return &signal{wg: wg}
}

// Notify triggers any waiters to complete. Subsequent waits do not block.
func (s *signal) Notify() {
	s.wg.Done()
}

// Wait waits for the signal to be notified before returning. The AsyncContext
// ensures this is invoked within an async context where blocking is acceptable.
func (s *signal) Wait(ctx jsutil.AsyncContext) {
	s.wg.Wait()
}

// New returns a new UI instance that manages keys using the supplied manager.
// domObj is the DOM instance corresponding to the document in which the Options
// UI is displayed.
func New(mgr keys.Manager, domObj *dom.Doc) *UI {
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
	cf.Add(dom.OnClick(result.addButton, result.add))
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
	dom.RemoveChildren(u.errorText)

	if err != nil {
		jsutil.LogError("UI.setError(): %v", err)
		dom.AppendChild(u.errorText, u.dom.NewText(err.Error()), nil)
	}
}

// add configures a new key.  It displays a dialog prompting the user for a name
// and the corresponding private key.  If the user continues, the key is
// added to the manager.
func (u *UI) add(ctx jsutil.AsyncContext, evt dom.Event) {
	ok, name, privateKey := u.promptAdd(ctx)
	if !ok {
		return
	}

	if err := u.mgr.Add(ctx, name, privateKey); err != nil {
		u.setError(fmt.Errorf("failed to add key: %v", err))
		return
	}

	u.setError(nil)
	u.updateKeys(ctx)
}

// promptAdd displays a dialog prompting the user for a name and private key.
func (u *UI) promptAdd(ctx jsutil.AsyncContext) (ok bool, name, privateKey string) {
	dialog := dom.NewDialog(u.dom.GetElement("addDialog"))
	form := u.dom.GetElement("addForm")
	nameField := u.dom.GetElement("addName")
	keyField := u.dom.GetElement("addKey")
	cancel := u.dom.GetElement("addCancel")

	sig := newSignal()
	var cleanup jsutil.CleanupFuncs
	cleanup.Add(dom.OnSubmit(form, func(ctx jsutil.AsyncContext, evt dom.Event) {
		ok = true
		name = dom.Value(nameField)
		privateKey = dom.Value(keyField)
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dom.OnClick(cancel, func(ctx jsutil.AsyncContext, evt dom.Event) {
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dialog.OnClose(func(ctx jsutil.AsyncContext, evt dom.Event) {
		dom.SetValue(nameField, "")
		dom.SetValue(keyField, "")
		cleanup.Do()
	}))

	dialog.ShowModal()
	sig.Wait(ctx)
	return
}

// load loads the key with the specified ID.  A dialog prompts the user for a
// passphrase if the private key is encrypted.
func (u *UI) load(ctx jsutil.AsyncContext, id keys.ID) {
	k := u.keyByID(id)
	if k == nil {
		u.setError(fmt.Errorf("failed to unload key ID %s: not found", id))
		return
	}

	var ok bool
	var passphrase string
	if k.Encrypted {
		ok, passphrase = u.promptPassphrase(ctx)
		if !ok {
			return
		}
	}

	if err := u.mgr.Load(ctx, id, passphrase); err != nil {
		u.setError(fmt.Errorf("failed to load key: %v", err))
		return
	}
	u.setError(nil)
	u.updateKeys(ctx)
}

// promptPassphrase displays a dialog prompting the user for a passphrase.
func (u *UI) promptPassphrase(ctx jsutil.AsyncContext) (ok bool, passphrase string) {
	dialog := dom.NewDialog(u.dom.GetElement("passphraseDialog"))
	form := u.dom.GetElement("passphraseForm")
	passphraseField := u.dom.GetElement("passphrase")
	cancel := u.dom.GetElement("passphraseCancel")

	sig := newSignal()
	var cleanup jsutil.CleanupFuncs
	cleanup.Add(dom.OnSubmit(form, func(ctx jsutil.AsyncContext, evt dom.Event) {
		ok = true
		passphrase = dom.Value(passphraseField)
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dom.OnClick(cancel, func(ctx jsutil.AsyncContext, evt dom.Event) {
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dialog.OnClose(func(ctx jsutil.AsyncContext, evt dom.Event) {
		dom.SetValue(passphraseField, "")
		cleanup.Do()
	}))

	dialog.ShowModal()
	sig.Wait(ctx)
	return
}

// unload unloads the specified key.
func (u *UI) unload(ctx jsutil.AsyncContext, id keys.ID) {
	if err := u.mgr.Unload(ctx, id); err != nil {
		u.setError(fmt.Errorf("failed to unload key ID %s: %v", id, err))
		return
	}
	u.setError(nil)
	u.updateKeys(ctx)
}

// promptRemove displays a dialog prompting the user to confirm that a key
// should be removed.
func (u *UI) promptRemove(ctx jsutil.AsyncContext, id keys.ID) (yes bool) {
	k := u.keyByID(id)
	if k == nil {
		u.setError(fmt.Errorf("failed to remove key ID %s: not found", id))
		return
	}

	dialog := dom.NewDialog(u.dom.GetElement("removeDialog"))
	form := u.dom.GetElement("removeForm")
	name := u.dom.GetElement("removeName")
	no := u.dom.GetElement("removeNo")
	dom.AppendChild(name, u.dom.NewText(k.Name), nil)

	sig := newSignal()
	var cleanup jsutil.CleanupFuncs
	cleanup.Add(dom.OnSubmit(form, func(ctx jsutil.AsyncContext, evt dom.Event) {
		yes = true
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dom.OnClick(no, func(ctx jsutil.AsyncContext, evt dom.Event) {
		dialog.Close()
		sig.Notify()
	}))
	cleanup.Add(dialog.OnClose(func(ctx jsutil.AsyncContext, evt dom.Event) {
		dom.RemoveChildren(name)
		cleanup.Do()
	}))

	dialog.ShowModal()
	sig.Wait(ctx)
	return
}

// remove removes the key with the specified ID.  A dialog prompts the user to
// confirm that the key should be removed.
func (u *UI) remove(ctx jsutil.AsyncContext, id keys.ID) {
	if yes := u.promptRemove(ctx, id); !yes {
		return
	}

	if err := u.mgr.Remove(ctx, id); err != nil {
		u.setError(fmt.Errorf("failed to remove key ID %s: %v", id, err))
		return
	}
	u.setError(nil)
	u.updateKeys(ctx)
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
	dom.RemoveChildren(u.keysData)
	for _, k := range u.keys {
		k.cleanup.Do()
	}

	// Construct elements for new keys.
	for _, k := range newKeys {
		k := k
		dom.AppendChild(u.keysData, u.dom.NewElement("tr"), func(row js.Value) {
			// Key name
			dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyName")
					dom.AppendChild(div, u.dom.NewText(k.Name), nil)
				})
			})

			// Controls
			dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyControls")
					if k.ID == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					if k.Loaded {
						// Unload button
						dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(UnloadButton, k.ID))
							dom.AppendChild(btn, u.dom.NewText("Unload"), nil)
							k.cleanup.Add(dom.OnClick(btn, func(ctx jsutil.AsyncContext, evt dom.Event) {
								u.unload(ctx, k.ID)
							}))
						})
					} else {
						// Load button
						dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
							btn.Set("type", "button")
							btn.Set("id", buttonID(LoadButton, k.ID))
							dom.AppendChild(btn, u.dom.NewText("Load"), nil)
							k.cleanup.Add(dom.OnClick(btn, func(ctx jsutil.AsyncContext, evt dom.Event) {
								u.load(ctx, k.ID)
							}))
						})
					}

					// Remove button
					dom.AppendChild(div, u.dom.NewElement("button"), func(btn js.Value) {
						btn.Set("type", "button")
						btn.Set("id", buttonID(RemoveButton, k.ID))
						dom.AppendChild(btn, u.dom.NewText("Remove"), nil)
						k.cleanup.Add(dom.OnClick(btn, func(ctx jsutil.AsyncContext, evt dom.Event) {
							u.remove(ctx, k.ID)
						}))
					})
				})
			})

			// Type
			dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyType")
					dom.AppendChild(div, u.dom.NewText(k.Type), nil)
				})
			})

			// Blob
			dom.AppendChild(row, u.dom.NewElement("td"), func(cell js.Value) {
				dom.AppendChild(cell, u.dom.NewElement("div"), func(div js.Value) {
					div.Set("className", "keyBlob")
					dom.AppendChild(div, u.dom.NewText(k.Blob), nil)
				})
			})
		})
	}
	// Update internal state after DOM is updated. Otherwise, callers (e.g.,
	// our end-to-end test) may look for the new DOM elements before they
	// are available.
	u.keys = newKeys
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
func (u *UI) updateKeys(ctx jsutil.AsyncContext) {
	configured, err := u.mgr.Configured(ctx)
	if err != nil {
		u.setError(fmt.Errorf("failed to get configured keys: %v", err))
		return
	}

	loaded, err := u.mgr.Loaded(ctx)
	if err != nil {
		u.setError(fmt.Errorf("failed to get loaded keys: %v", err))
		return
	}
	u.setError(nil)
	u.setKeys(mergeKeys(configured, loaded))
}

const (
	pollInterval = 100 * time.Millisecond
	pollTimeout  = 10 * time.Second
)

// poll checks a condition for up to a timeout.
//
// The AsyncContext ensures this is invoked within an async context where
// blocking is acceptable.
func poll(ctx jsutil.AsyncContext, done func() bool) bool {
	timeout := time.Now().Add(pollTimeout)
	for time.Now().Before(timeout) {
		if done() {
			return true
		}
		time.Sleep(pollInterval)
	}
	return false
}

// EndToEndTest runs a set of tests via the UI.  Failures are returned as a list
// of errors.
//
// No attempt is made to clean up from any intermediate state should the test
// fail.
func (u *UI) EndToEndTest(ctx jsutil.AsyncContext) []error {
	jsutil.Log("Starting test")
	defer jsutil.Log("Finished test")

	addDialog := u.dom.GetElement("addDialog")
	addButton := u.dom.GetElement("add")
	addName := u.dom.GetElement("addName")
	addKey := u.dom.GetElement("addKey")
	addOk := u.dom.GetElement("addOk")
	passphraseDialog := u.dom.GetElement("passphraseDialog")
	passphraseInput := u.dom.GetElement("passphrase")
	passphraseOk := u.dom.GetElement("passphraseOk")
	removeDialog := u.dom.GetElement("removeDialog")
	removeYes := u.dom.GetElement("removeYes")

	var errs []error

	jsutil.Log("Generate random name to use for key")
	i, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to generate random number: %v", err))
		return errs
	}
	keyName := fmt.Sprintf("e2e-test-key-%s", i.String())

	jsutil.Log("Configure a new key")
	dom.DoClick(addButton)
	if !poll(ctx, func() bool { return addDialog.Get("open").Bool() }) {
		errs = append(errs, fmt.Errorf("add dialog failed to open"))
		return errs
	}
	dom.SetValue(addName, keyName)
	// Use the long key to exercise storage of large values in Chrome storage.
	dom.SetValue(addKey, testdata.LongKeyWithPassphrase.Private)
	dom.DoClick(addOk)

	jsutil.Log("Validate configured keys; ensure new key is present")
	var key *displayedKey
	if !poll(ctx, func() bool {
		key = u.keyByName(keyName)
		return key != nil
	}) {
		errs = append(errs, fmt.Errorf("after added: failed to find key"))
		return errs
	}

	jsutil.Log("Load the new key")
	dom.DoClick(u.dom.GetElement(buttonID(LoadButton, key.ID)))
	if !poll(ctx, func() bool { return passphraseDialog.Get("open").Bool() }) {
		errs = append(errs, fmt.Errorf("passphrase dialog failed to open"))
		return errs
	}
	dom.SetValue(passphraseInput, testdata.LongKeyWithPassphrase.Passphrase)
	dom.DoClick(passphraseOk)

	jsutil.Log("Validate loaded keys; ensure new key is loaded")
	if !poll(ctx, func() bool {
		key = u.keyByName(keyName)
		return key != nil && key.Loaded
	}) {
		errs = append(errs, fmt.Errorf("after loaded: failed to find loaded key"))
		return errs
	}
	if diff := cmp.Diff(key.Loaded, true); diff != "" {
		errs = append(errs, fmt.Errorf("after load: incorrect loaded state: %s", diff))
	}
	if diff := cmp.Diff(key.Type, testdata.LongKeyWithPassphrase.Type); diff != "" {
		errs = append(errs, fmt.Errorf("after load: incorrect type: %s", diff))
	}
	if diff := cmp.Diff(key.Blob, testdata.LongKeyWithPassphrase.Blob); diff != "" {
		errs = append(errs, fmt.Errorf("after load: incorrect blob: %s", diff))
	}

	jsutil.Log("Unload key")
	dom.DoClick(u.dom.GetElement(buttonID(UnloadButton, key.ID)))

	jsutil.Log("Validate loaded keys; ensure key is unloaded")
	if !poll(ctx, func() bool {
		key = u.keyByName(keyName)
		return key != nil && !key.Loaded
	}) {
		errs = append(errs, fmt.Errorf("after unload: failed to find unloaded key"))
		return errs // Remaining tests have hard dependency on unloaded key.
	}
	if diff := cmp.Diff(key.Loaded, false); diff != "" {
		errs = append(errs, fmt.Errorf("after unload: incorrect loaded state: %s", diff))
	}
	if diff := cmp.Diff(key.Type, ""); diff != "" {
		errs = append(errs, fmt.Errorf("after unload: incorrect type: %s", diff))
	}
	if diff := cmp.Diff(key.Blob, ""); diff != "" {
		errs = append(errs, fmt.Errorf("after unload: incorrect blob: %s", diff))
	}

	jsutil.Log("Remove key")
	dom.DoClick(u.dom.GetElement(buttonID(RemoveButton, key.ID)))
	if !poll(ctx, func() bool { return removeDialog.Get("open").Bool() }) {
		errs = append(errs, fmt.Errorf("remove dialog failed to open"))
		return errs
	}
	dom.DoClick(removeYes)

	jsutil.Log("Validate configured keys; ensure key is removed")
	if !poll(ctx, func() bool {
		key = u.keyByName(keyName)
		return key == nil
	}) {
		errs = append(errs, fmt.Errorf("after removed: failed to observe key as removed"))
		return errs
	}

	return errs

}
