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

	loadedList = doc.Call("getElementById", "loadedKeys")

	availableList   = doc.Call("getElementById", "availableKeys")
	availableAdd    = doc.Call("getElementById", "add")
	availableRemove = doc.Call("getElementById", "remove")
	availableLoad   = doc.Call("getElementById", "load")

	addDialog = doc.Call("getElementById", "addDialog")
	addName   = doc.Call("getElementById", "addName")
	addKey    = doc.Call("getElementById", "addKey")
	addOk     = doc.Call("getElementById", "addOk")
	addCancel = doc.Call("getElementById", "addCancel")

	errorText = doc.Call("getElementById", "errorMessage")
)

func nodeListToArray(o *js.Object) []*js.Object {
	var result []*js.Object
	length := o.Get("length").Int()
	for i := 0; i < length; i++ {
		result = append(result, o.Call("item", i))
	}
	return result
}

func selectedValues(o *js.Object) []string {
	var result []string
	for _, s := range nodeListToArray(availableList.Get("selectedOptions")) {
		result = append(result, s.Get("value").String())
	}
	return result
}

func removeChildren(l *js.Object) {
	for l.Call("hasChildNodes").Bool() {
		l.Call("removeChild", l.Get("firstChild"))
	}
}

func updateSelectList(l *js.Object, elements []string) {
	removeChildren(l)
	for _, e := range elements {
		opt := doc.Call("createElement", "option")
		opt.Set("text", e)
		l.Call("appendChild", opt)
	}
}

func updateLoadedKeys(avail keys.Available) {
	avail.Loaded(func(keys []string, err error) {
		if err != nil {
			setError(fmt.Errorf("failed to read loaded keys: %v", err))
			return
		}

		setError(nil)
		updateSelectList(loadedList, keys)
	})
}

func updateAvailableKeys(avail keys.Available) {
	avail.Available(func(keys []string, err error) {
		if err != nil {
			setError(fmt.Errorf("failed to read available keys: %v", err))
			return
		}

		setError(nil)
		updateSelectList(availableList, keys)
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
		updateLoadedKeys(avail)
		updateAvailableKeys(avail)
	})

	// Add new key
	availableAdd.Call("addEventListener", "click", func() {
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
				updateAvailableKeys(avail)
			})
		})
	})

	// Remove selected keys
	availableRemove.Call("addEventListener", "click", func() {
		for _, val := range selectedValues(availableList) {
			avail.Remove(val, func(err error) {
				if err != nil {
					setError(fmt.Errorf("failed to remove key %s: %v", val, err))
					return
				}

				setError(nil)
				updateAvailableKeys(avail)
			})
		}
	})

	// Load a key.
	availableLoad.Call("addEventListener", "click", func() {
		for _, val := range selectedValues(availableList) {
			promptPassphrase(func(passphrase string, ok bool) {
				if !ok {
					return
				}
				avail.Load(val, passphrase, func(err error) {
					if err != nil {
						setError(fmt.Errorf("failed to load key: %v", err))
						return
					}
					setError(nil)
					updateLoadedKeys(avail)
				})
			})
		}
	})
}
