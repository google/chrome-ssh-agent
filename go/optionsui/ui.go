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
	"encoding/base64"
	"fmt"
	"log"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/google/chrome-ssh-agent/go/keys"
	"github.com/gopherjs/gopherjs/js"
)

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
	errorText        *js.Object
	keysData         *js.Object
	keys             []*displayedKey
}

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
		errorText:        domObj.GetElement("errorMessage"),
		keysData:         domObj.GetElement("keysData"),
	}
	result.load()
	return result
}

func (u *UI) load() {
	// Populate keys on initial display
	u.dom.OnDOMContentLoaded(u.updateKeys)
	// Configure new key on click
	u.dom.OnClick(u.addButton, u.Add)
}

func (u *UI) setError(err error) {
	// Clear any existing error
	u.dom.RemoveChildren(u.errorText)

	if err != nil {
		u.dom.AppendChild(u.errorText, u.dom.NewText(err.Error()), nil)
	}
}

func (u *UI) Add() {
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

func (u *UI) promptAdd(callback func(name, privateKey string, ok bool)) {
	u.dom.OnClick(u.addOk, func() {
		n := u.dom.Value(u.addName)
		k := u.dom.Value(u.addKey)
		u.dom.SetValue(u.addName, "")
		u.dom.SetValue(u.addKey, "")
		u.dom.Close(u.addDialog)
		callback(n, k, true)
	})
	u.dom.OnClick(u.addCancel, func() {
		u.dom.SetValue(u.addName, "")
		u.dom.SetValue(u.addKey, "")
		u.dom.Close(u.addDialog)
		callback("", "", false)
	})
	u.dom.ShowModal(u.addDialog)
}

func (u *UI) Load(id keys.ID) {
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

func (u *UI) promptPassphrase(callback func(passphrase string, ok bool)) {
	u.dom.OnClick(u.passphraseOk, func() {
		p := u.dom.Value(u.passphraseInput)
		u.dom.SetValue(u.passphraseInput, "")
		u.dom.Close(u.passphraseDialog)
		callback(p, true)
	})
	u.dom.OnClick(u.passphraseCancel, func() {
		u.dom.SetValue(u.passphraseInput, "")
		u.dom.Close(u.passphraseDialog)
		callback("", false)
	})
	u.dom.ShowModal(u.passphraseDialog)
}

func (u *UI) Remove(id keys.ID) {
	u.mgr.Remove(id, func(err error) {
		if err != nil {
			u.setError(fmt.Errorf("failed to remove key: %v", err))
			return
		}

		u.setError(nil)
		u.updateKeys()
	})
}

type displayedKey struct {
	Id     keys.ID
	Loaded bool
	Name   string
	Type   string
	Blob   string
}

func (u *UI) DisplayedKeys() []*displayedKey {
	return u.keys
}

type buttonKind int

const (
	LoadButton buttonKind = iota
	RemoveButton
)

func buttonId(kind buttonKind, id keys.ID) string {
	s := "unknown"
	switch kind {
	case LoadButton:
		s = "load"
	case RemoveButton:
		s = "remove"
	}
	return fmt.Sprintf("%s-%s", s, id)
}

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
					if k.Id == keys.InvalidID {
						// We only control keys with a valid ID.
						return
					}

					// Load button
					if !k.Loaded {
						u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn *js.Object) {
							btn.Set("type", "button")
							btn.Set("id", buttonId(LoadButton, k.Id))
							u.dom.AppendChild(btn, u.dom.NewText("Load"), nil)
							u.dom.OnClick(btn, func() {
								u.Load(k.Id)
							})
						})
					}

					// Remove button
					u.dom.AppendChild(div, u.dom.NewElement("button"), func(btn *js.Object) {
						btn.Set("type", "button")
						btn.Set("id", buttonId(RemoveButton, k.Id))
						log.Printf("created button with id: %s", buttonId(RemoveButton, k.Id))
						u.dom.AppendChild(btn, u.dom.NewText("Remove"), nil)
						u.dom.OnClick(btn, func() {
							u.Remove(k.Id)
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
