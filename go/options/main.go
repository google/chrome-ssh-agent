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

package main

import (
	"encoding/base64"
	"fmt"

	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/gopherjs/gopherjs/js"
)

var (
	doc = js.Global.Get("document")

	passphraseDialog = doc.Call("getElementById", "passphraseDialog")
	passphraseInput  = doc.Call("getElementById", "passphrase")
	passphraseOk     = doc.Call("getElementById", "passphraseOk")
	passphraseCancel = doc.Call("getElementById", "passphraseCancel")

	addButton = doc.Call("getElementById", "add")
	addDialog = doc.Call("getElementById", "addDialog")
	addName   = doc.Call("getElementById", "addName")
	addKey    = doc.Call("getElementById", "addKey")
	addOk     = doc.Call("getElementById", "addOk")
	addCancel = doc.Call("getElementById", "addCancel")

	errorText = doc.Call("getElementById", "errorMessage")

	keysData = doc.Call("getElementById", "keysData")
)

type displayedKey struct {
	Id     keys.ID
	Loaded bool
	Name   string
	Type   string
	Blob   string
}

func removeChildren(l *js.Object) {
	for l.Call("hasChildNodes").Bool() {
		l.Call("removeChild", l.Get("firstChild"))
	}
}

func newElement(kind string) *js.Object {
	return doc.Call("createElement", kind)
}

func newText(text string) *js.Object {
	return doc.Call("createTextNode", text)
}

func appendChild(parent, child *js.Object, populate func(child *js.Object)) {
	if populate != nil {
		populate(child)
	}
	parent.Call("appendChild", child)
}

func loadKey(avail keys.Available, id keys.ID) {
	promptPassphrase(func(passphrase string, ok bool) {
		if !ok {
			return
		}
		avail.Load(id, passphrase, func(err error) {
			if err != nil {
				setError(fmt.Errorf("failed to load key: %v", err))
				return
			}
			setError(nil)
			updateKeys(avail)
		})
	})
}

func removeKey(avail keys.Available, id keys.ID) {
	avail.Remove(id, func(err error) {
		if err != nil {
			setError(fmt.Errorf("failed to remove key: %v", err))
			return
		}

		setError(nil)
		updateKeys(avail)
	})
}

func setDisplayedKeys(avail keys.Available, displayed []*displayedKey) {
	removeChildren(keysData)

	for _, k := range displayed {
		k := k
		appendChild(keysData, newElement("tr"), func(row *js.Object) {
			// Key name
			appendChild(row, newElement("td"), func(cell *js.Object) {
				appendChild(cell, newElement("div"), func(div *js.Object) {
					div.Set("className", "keyName")
					appendChild(div, newText(k.Name), nil)
				})
			})

			// Controls
			appendChild(row, newElement("td"), func(cell *js.Object) {
				appendChild(cell, newElement("div"), func(div *js.Object) {
					div.Set("className", "keyControls")
					if k.Id == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					// Load button
					if !k.Loaded {
						appendChild(div, newElement("button"), func(btn *js.Object) {
							btn.Set("type", "button")
							appendChild(btn, newText("Load"), nil)
							btn.Call("addEventListener", "click", func() {
								loadKey(avail, k.Id)
							})
						})
					}

					// Remove button
					appendChild(div, newElement("button"), func(btn *js.Object) {
						btn.Set("type", "button")
						appendChild(btn, newText("Remove"), nil)
						btn.Call("addEventListener", "click", func() {
							removeKey(avail, k.Id)
						})
					})
				})
			})

			// Type
			appendChild(row, newElement("td"), func(cell *js.Object) {
				appendChild(cell, newElement("div"), func(div *js.Object) {
					div.Set("className", "keyType")
					appendChild(div, newText(k.Type), nil)
				})
			})

			// Blob
			appendChild(row, newElement("td"), func(cell *js.Object) {
				appendChild(cell, newElement("div"), func(div *js.Object) {
					div.Set("className", "keyBlob")
					appendChild(div, newText(k.Blob), nil)
				})
			})
		})
	}
}

func mergeKeys(available []*keys.Key, loaded []*keys.LoadedKey) []*displayedKey {
	// Build map of available keys for faster lookup
	availableMap := make(map[keys.ID]*keys.Key)
	for _, k := range available {
		availableMap[k.Id] = k
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
			Blob:   base64.StdEncoding.EncodeToString([]byte(l.Blob)),
		}
		// Attempt to figure out if this is a key we loaded. If so, fill
		// in some additional information.  It is possible that a key with
		// a non-existent ID is loaded (e.g., it was removed while loaded);
		// in this case we claim we do not have an ID.
		if id := keys.GetID(l); id != keys.InvalidID {
			if ak := availableMap[id]; ak != nil {
				loadedIds[id] = true
				dk.Id = id
				dk.Name = ak.Name
			}
		}
		result = append(result, dk)
	}

	// Add all available keys that are not loaded.
	for _, a := range available {
		// Skip any that we already covered above.
		if loadedIds[a.Id] {
			continue
		}

		result = append(result, &displayedKey{
			Id:     a.Id,
			Loaded: false,
			Name:   a.Name,
		})
	}

	// TODO(ralimi) Sort displayed items to ensure consitent ordering over time

	return result
}

func updateKeys(avail keys.Available) {
	avail.Available(func(available []*keys.Key, err error) {
		if err != nil {
			setError(fmt.Errorf("failed to get available keys: %v", err))
			return
		}

		avail.Loaded(func(loaded []*keys.LoadedKey, err error) {
			if err != nil {
				setError(fmt.Errorf("failed to get loaded keys: %v", err))
				return
			}

			setError(nil)
			setDisplayedKeys(avail, mergeKeys(available, loaded))
		})
	})
}

func promptAdd(callback func(name, privateKey string, ok bool)) {
	addOk.Call("addEventListener", "click", func() {
		n := addName.Get("value").String()
		k := addKey.Get("value").String()
		addName.Set("value", "")
		addKey.Set("value", "")
		addDialog.Call("close")
		callback(n, k, true)
	})
	addCancel.Call("addEventListener", "click", func() {
		addName.Set("value", "")
		addKey.Set("value", "")
		addDialog.Call("close")
		callback("", "", false)
	})
	addDialog.Call("showModal")
}

func promptPassphrase(callback func(passphrase string, ok bool)) {
	passphraseOk.Call("addEventListener", "click", func() {
		p := passphraseInput.Get("value").String()
		passphraseInput.Set("value", "")
		passphraseDialog.Call("close")
		callback(p, true)
	})
	passphraseCancel.Call("addEventListener", "click", func() {
		passphraseInput.Set("value", "")
		passphraseDialog.Call("close")
		callback("", false)
	})
	passphraseDialog.Call("showModal")
}

func setError(err error) {
	// Clear any existing error
	removeChildren(errorText)

	if err != nil {
		errorText.Call("appendChild", doc.Call("createTextNode", err.Error()))
	}
}

func main() {
	avail := keys.NewClient()

	// Load settings on initial display
	doc.Call("addEventListener", "DOMContentLoaded", func() {
		updateKeys(avail)
	})

	// Add new key
	addButton.Call("addEventListener", "click", func() {
		promptAdd(func(name, privateKey string, ok bool) {
			if !ok {
				return
			}
			avail.Add(name, privateKey, func(err error) {
				if err != nil {
					setError(fmt.Errorf("failed to add key: %v", err))
					return
				}

				setError(nil)
				updateKeys(avail)
			})
		})
	})
}
