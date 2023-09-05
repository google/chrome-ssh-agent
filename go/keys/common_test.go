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
	"sort"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func findKey(ctx jsutil.AsyncContext, mgr Manager, byID ID, byName string) (ID, error) {
	if byID != InvalidID {
		return byID, nil
	}

	configured, err := mgr.Configured(ctx)
	if err != nil {
		return InvalidID, err
	}

	for _, k := range configured {
		if k.Name == byName {
			return ID(k.ID), nil
		}
	}

	return InvalidID, fmt.Errorf("failed to find key with name %s", byName)
}

func configuredKeyNames(keys []*ConfiguredKey) []string {
	var result []string
	for _, k := range keys {
		result = append(result, k.Name)
	}
	sort.Strings(result)
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
		result = append(result, base64.StdEncoding.EncodeToString(k.Blob()))
	}
	return result
}

func loadedKeyIDs(keys []*LoadedKey) []ID {
	var res []ID
	for _, k := range keys {
		res = append(res, k.ID())
	}
	return res
}

func sessionKeyIDs(ctx jsutil.AsyncContext, sessionKeys *storage.Typed[sessionKey]) ([]ID, error) {
	keys, err := sessionKeys.ReadAll(ctx)

	var res []ID
	for _, k := range keys {
		res = append(res, ID(k.ID))
	}
	return res, err
}

var (
	// Custom Comparer for LoadedKey type.  'blob' is an unexported
	// field, so we explicitly compare it.
	loadedKeyCmp = cmp.Comparer(func(a, b *LoadedKey) bool {
		return cmp.Equal(a, b, cmpopts.IgnoreUnexported(LoadedKey{})) &&
			cmp.Equal(a.Blob(), b.Blob())
	})

	// Custom Comparers for errors. Used only when we can't use
	// the standard cmpopts.EquateErrors. Usage of these comparers
	// should document why cmpopts.EquateErrors does not suffice.
	errStringCmp = cmp.Comparer(func(a, b error) bool {
		return a.Error() == b.Error()
	})

	// Option for order-independent slices of IDs.
	idSlice = cmpopts.SortSlices(func(a, b ID) bool {
		return a < b
	})
)
