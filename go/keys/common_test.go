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

package keys

import (
	"encoding/base64"
	"fmt"
)

func syncAdd(mgr Manager, name string, pemPrivateKey string) error {
	errc := make(chan error, 1)
	mgr.Add(name, pemPrivateKey, func(err error) {
		errc <- err
		close(errc)
	})
	return readErr(errc)
}

func syncRemove(mgr Manager, id ID) error {
	errc := make(chan error, 1)
	mgr.Remove(id, func(err error) {
		errc <- err
		close(errc)
	})
	return readErr(errc)
}

func syncConfigured(mgr Manager) ([]*ConfiguredKey, error) {
	errc := make(chan error, 1)
	var result []*ConfiguredKey
	mgr.Configured(func(keys []*ConfiguredKey, err error) {
		result = keys
		errc <- err
		close(errc)
	})
	err := readErr(errc)
	return result, err
}

func syncLoad(mgr Manager, id ID, passphrase string) error {
	errc := make(chan error, 1)
	mgr.Load(id, passphrase, func(err error) {
		errc <- err
		close(errc)
	})
	return readErr(errc)
}

func syncLoaded(mgr Manager) ([]*LoadedKey, error) {
	errc := make(chan error, 1)
	var result []*LoadedKey
	mgr.Loaded(func(keys []*LoadedKey, err error) {
		result = keys
		errc <- err
		close(errc)
	})
	err := readErr(errc)
	return result, err
}

func readErr(errc chan error) error {
	for err := range errc {
		return err
	}
	panic("no elements read from channel")
}

func findKey(mgr Manager, byID ID, byName string) (ID, error) {
	if byID != InvalidID {
		return byID, nil
	}

	configured, err := syncConfigured(mgr)
	if err != nil {
		return InvalidID, err
	}

	for _, k := range configured {
		if k.Name == byName {
			return k.ID, nil
		}
	}

	return InvalidID, fmt.Errorf("failed to find key with name %s", byName)
}

func configuredKeyNames(keys []*ConfiguredKey) []string {
	var result []string
	for _, k := range keys {
		result = append(result, k.Name)
	}
	return result
}

func loadedKeyIds(keys []*LoadedKey) []ID {
	var result []ID
	for _, k := range keys {
		result = append(result, k.ID())
	}
	return result
}

func loadedKeyBlobs(keys []*LoadedKey) []string {
	var result []string
	for _, k := range keys {
		result = append(result, base64.StdEncoding.EncodeToString([]byte(k.Blob)))
	}
	return result
}
