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

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/gopherjs/gopherjs/js"
)

var (
	passphraseDialog = dom.GetElement("passphraseDialog")
	passphraseInput  = dom.GetElement("passphrase")
	passphraseOk     = dom.GetElement("passphraseOk")
	passphraseCancel = dom.GetElement("passphraseCancel")

	addButton = dom.GetElement("add")
	addDialog = dom.GetElement("addDialog")
	addName   = dom.GetElement("addName")
	addKey    = dom.GetElement("addKey")
	addOk     = dom.GetElement("addOk")
	addCancel = dom.GetElement("addCancel")

	errorText = dom.GetElement("errorMessage")

	keysData = dom.GetElement("keysData")
)

type displayedKey struct {
	Id     keys.ID
	Loaded bool
	Name   string
	Type   string
	Blob   string
}

func loadKey(mgr keys.Manager, id keys.ID) {
	promptPassphrase(func(passphrase string, ok bool) {
		if !ok {
			return
		}
		mgr.Load(id, passphrase, func(err error) {
			if err != nil {
				setError(fmt.Errorf("failed to load key: %v", err))
				return
			}
			setError(nil)
			updateKeys(mgr)
		})
	})
}

func removeKey(mgr keys.Manager, id keys.ID) {
	mgr.Remove(id, func(err error) {
		if err != nil {
			setError(fmt.Errorf("failed to remove key: %v", err))
			return
		}

		setError(nil)
		updateKeys(mgr)
	})
}

func setDisplayedKeys(mgr keys.Manager, displayed []*displayedKey) {
	dom.RemoveChildren(keysData)

	for _, k := range displayed {
		k := k
		dom.AppendChild(keysData, dom.NewElement("tr"), func(row *js.Object) {
			// Key name
			dom.AppendChild(row, dom.NewElement("td"), func(cell *js.Object) {
				dom.AppendChild(cell, dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyName")
					dom.AppendChild(div, dom.NewText(k.Name), nil)
				})
			})

			// Controls
			dom.AppendChild(row, dom.NewElement("td"), func(cell *js.Object) {
				dom.AppendChild(cell, dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyControls")
					if k.Id == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					// Load button
					if !k.Loaded {
						dom.AppendChild(div, dom.NewElement("button"), func(btn *js.Object) {
							btn.Set("type", "button")
							dom.AppendChild(btn, dom.NewText("Load"), nil)
							dom.OnClick(btn, func() {
								loadKey(mgr, k.Id)
							})
						})
					}

					// Remove button
					dom.AppendChild(div, dom.NewElement("button"), func(btn *js.Object) {
						btn.Set("type", "button")
						dom.AppendChild(btn, dom.NewText("Remove"), nil)
						dom.OnClick(btn, func() {
							removeKey(mgr, k.Id)
						})
					})
				})
			})

			// Type
			dom.AppendChild(row, dom.NewElement("td"), func(cell *js.Object) {
				dom.AppendChild(cell, dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyType")
					dom.AppendChild(div, dom.NewText(k.Type), nil)
				})
			})

			// Blob
			dom.AppendChild(row, dom.NewElement("td"), func(cell *js.Object) {
				dom.AppendChild(cell, dom.NewElement("div"), func(div *js.Object) {
					div.Set("className", "keyBlob")
					dom.AppendChild(div, dom.NewText(k.Blob), nil)
				})
			})
		})
	}
}

func mergeKeys(configured []*keys.ConfiguredKey, loaded []*keys.LoadedKey) []*displayedKey {
	// Build map of configured keys for faster lookup
	configuredMap := make(map[keys.ID]*keys.ConfiguredKey)
	for _, k := range configured {
		configuredMap[k.Id] = k
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
		if id := l.ID(); id != keys.InvalidID {
			if ak := configuredMap[id]; ak != nil {
				loadedIds[id] = true
				dk.Id = id
				dk.Name = ak.Name
			}
		}
		result = append(result, dk)
	}

	// Add all configured keys that are not loaded.
	for _, a := range configured {
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

func updateKeys(mgr keys.Manager) {
	mgr.Configured(func(configured []*keys.ConfiguredKey, err error) {
		if err != nil {
			setError(fmt.Errorf("failed to get configured keys: %v", err))
			return
		}

		mgr.Loaded(func(loaded []*keys.LoadedKey, err error) {
			if err != nil {
				setError(fmt.Errorf("failed to get loaded keys: %v", err))
				return
			}

			setError(nil)
			setDisplayedKeys(mgr, mergeKeys(configured, loaded))
		})
	})
}

func promptAdd(callback func(name, privateKey string, ok bool)) {
	dom.OnClick(addOk, func() {
		n := addName.Get("value").String()
		k := addKey.Get("value").String()
		addName.Set("value", "")
		addKey.Set("value", "")
		addDialog.Call("close")
		callback(n, k, true)
	})
	dom.OnClick(addCancel, func() {
		addName.Set("value", "")
		addKey.Set("value", "")
		addDialog.Call("close")
		callback("", "", false)
	})
	addDialog.Call("showModal")
}

func promptPassphrase(callback func(passphrase string, ok bool)) {
	dom.OnClick(passphraseOk, func() {
		p := passphraseInput.Get("value").String()
		passphraseInput.Set("value", "")
		passphraseDialog.Call("close")
		callback(p, true)
	})
	dom.OnClick(passphraseCancel, func() {
		passphraseInput.Set("value", "")
		passphraseDialog.Call("close")
		callback("", false)
	})
	passphraseDialog.Call("showModal")
}

func setError(err error) {
	// Clear any existing error
	dom.RemoveChildren(errorText)

	if err != nil {
		dom.AppendChild(errorText, dom.NewText(err.Error()), nil)
	}
}

func main() {
	mgr := keys.NewClient()

	// Load settings on initial display
	dom.OnDOMContentLoaded(func() {
		updateKeys(mgr)
	})

	// Add new key
	dom.OnClick(addButton, func() {
		promptAdd(func(name, privateKey string, ok bool) {
			if !ok {
				return
			}
			mgr.Add(name, privateKey, func(err error) {
				if err != nil {
					setError(fmt.Errorf("failed to add key: %v", err))
					return
				}

				setError(nil)
				updateKeys(mgr)
			})
		})
	})
}
