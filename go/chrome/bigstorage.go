//go:build js && wasm

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

package chrome

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/dom"
	"github.com/norunners/vert"
)

// BigStorage supports storing and retrieving keys and values of arbitrary sizes
// in persistent storage.  Items that fit within the per-item quota are stored
// normally; larger ones are split into multiple chunks.
//
// Overall storage quota still applies, but this bypasses the per-item quotas.
//
// BigStorage implements the PersistentStore interface.
type BigStorage struct {
	// maxItemBytes is the maximum size of the key and value for
	// an entry in the storage.
	maxItemBytes int

	// s is the underlying persistent storage.
	s PersistentStore
}

// bigValueManifest is the value stored in place of a big value.  It contains
// pointers to the chunks that actually contain the values.
type bigValueManifest struct {
	// Magic is a string that must equal manifestMagic. This distinguishes
	// our manifest from any other value that may have been stored.
	Magic string `js:"magic"`

	// ChunkKeys is the sequence of keys that are chunks of the stored
	// data for this value.
	ChunkKeys []string `js:"chunkKeys"`
}

func newBigValueManifest() *bigValueManifest {
	return &bigValueManifest{
		Magic: bigValueManifestMagic,
	}
}

func (b *bigValueManifest) Valid() bool {
	return b.Magic == bigValueManifestMagic
}

const (
	// bigValueManifestMagic is the magic string that we encode in manifests.
	bigValueManifestMagic = "3cc36853-b864-4122-beaa-516aa24448f6"

	// chunkKeyPrefix is the prefix added to keys for individual chunks.
	chunkKeyPrefix = "chunk-" + bigValueManifestMagic + ":"
)

var (
	// chunkKeyLength is the size of chunk keys that we store.
	chunkKeyLength = len(makeChunkKey("dummy"))
)

func (b *BigStorage) maxChunkSize() int {
	// Chunk values are stored as strings; subtract 2 to leave room for
	// quotes added as part of stringification when storing.
	res := b.maxItemBytes - chunkKeyLength - 2
	if res <= 0 {
		panic(fmt.Errorf("maxItemBytes=%d is insufficient; chunkKeyLength=%d", b.maxItemBytes, chunkKeyLength))
	}
	return res
}

func (b *BigStorage) canStore(key string, valJSON string) bool {
	// Values are stored as strings after converting to JSON.
	return len(key)+len(valJSON) <= b.maxItemBytes
}

// makeChunkKey returns the key at which this chunk should be stored.
func makeChunkKey(chunk string) string {
	h := sha256.Sum256([]byte(chunk))
	e := base64.StdEncoding.EncodeToString([]byte(h[:]))
	return chunkKeyPrefix + e
}

// isChunkKey detects if the key refers to a chunk.
func isChunkKey(key string) bool {
	return strings.HasPrefix(key, chunkKeyPrefix)
}

// See PersistentStore.Set().
func (b *BigStorage) Set(data map[string]js.Value, callback func(err error)) {
	maxChunkSize := b.maxChunkSize()

	chunked := map[string]js.Value{}
	for k, v := range data {
		json := dom.ToJSON(v)
		if b.canStore(k, json) {
			// Store directly. Value is small enough.
			chunked[k] = v
			continue
		}

		// Value too large. Break into chunks.
		manifest := newBigValueManifest()
		for i := 0; i < len(json); i += maxChunkSize {
			extent := i + maxChunkSize
			if extent > len(json) {
				extent = len(json)
			}

			// Key is the hash of the contents. This is a simple way
			// to avoid overwriting data.
			chunk := json[i:extent]
			chunkKey := makeChunkKey(chunk)

			// Add to manifest and data we will store.
			manifest.ChunkKeys = append(manifest.ChunkKeys, chunkKey)
			chunked[chunkKey] = js.ValueOf(chunk)
		}

		// Associate the manifest with the original key.
		chunked[k] = vert.ValueOf(manifest).JSValue()
	}

	b.s.Set(chunked, callback)
}

// See PersistentStore.Get().
func (b *BigStorage) Get(callback func(data map[string]js.Value, err error)) {
	b.s.Get(func(data map[string]js.Value, err error) {
		if err != nil {
			callback(nil, err)
			return
		}

		unchunked := map[string]js.Value{}
		for k, v := range data {
			if isChunkKey(k) {
				// Skip chunks. We read these as part of the manifest.
				continue
			}

			// Attempt to read as a manifest.
			var manifest bigValueManifest
			if err := vert.ValueOf(v).AssignTo(&manifest); err == nil && manifest.Valid() {
				// Concatenate chunks and parse the JSON.
				var json strings.Builder
				for _, chunkKey := range manifest.ChunkKeys {
					chunkVal, present := data[chunkKey]
					if !present {
						callback(nil, fmt.Errorf("failed to read data; chunk key %s missing", chunkKey))
						return
					}
					json.WriteString(chunkVal.String())
				}

				unchunked[k] = dom.FromJSON(json.String())
				continue
			}

			// This is just a simple key.
			unchunked[k] = v
		}

		callback(unchunked, nil)
	})
}

// See PersistentStore.Delete().
func (b *BigStorage) Delete(keys []string, callback func(err error)) {
	// Delete the requested keys.
	b.s.Delete(keys, func(err error) {
		if err != nil {
			callback(err)
			return
		}

		// Once successful, delete all chunks that are no longer
		// referenced by any manifest. This takes care of those that
		// were just deleted, as well as any dangling ones that may
		// have been left over from before.
		b.s.Get(func(data map[string]js.Value, err error) {
			if err != nil {
				callback(fmt.Errorf("failed to query for dangling chunks: %v", err))
				return
			}

			// Initially, consider all chunk keys as dangling.
			danglingChunkKeys := map[string]bool{}
			for k := range data {
				if isChunkKey(k) {
					danglingChunkKeys[k] = true
				}
			}

			// Remove those that are referenced by a manifest.
			for _, v := range data {
				var manifest bigValueManifest
				if err := vert.ValueOf(v).AssignTo(&manifest); err != nil || !manifest.Valid() {
					continue // This is not a manifest.
				}
				for _, chunkKey := range manifest.ChunkKeys {
					delete(danglingChunkKeys, chunkKey)
				}
			}

			// Delete dangling chunk keys.
			var dangling []string
			for k := range danglingChunkKeys {
				dangling = append(dangling, k)
			}
			b.s.Delete(dangling, func(err error) {
				if err != nil {
					callback(fmt.Errorf("failed to delete dangling chunks: %v", err))
					return
				}
				callback(nil)
			})
		})
	})
}
