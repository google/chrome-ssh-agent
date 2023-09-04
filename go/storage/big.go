//go:build js

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

package storage

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/google/chrome-ssh-agent/go/jsutil"
	"github.com/google/chrome-ssh-agent/go/lock"
	"github.com/norunners/vert"
)

// Big supports storing and retrieving keys and values of arbitrary sizes. Items
// that fit within the per-item quota are stored normally; larger ones are split
// into multiple chunks.
//
// Overall storage quota still applies, but this bypasses the per-item quotas.
//
// Big implements the Area interface.
type Big struct {
	// maxItemBytes is the maximum size of the key and value for
	// an entry in the storage.
	maxItemBytes int

	// s is the underlying storage area.
	s Area
}

func NewBig(maxItemBytes int, store Area) *Big {
	return &Big{
		maxItemBytes: maxItemBytes,
		s:            store,
	}
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

func (b *Big) maxChunkSize() int {
	// Chunk values are stored as strings; subtract 2 to leave room for
	// quotes added as part of stringification when storing.
	res := b.maxItemBytes - chunkKeyLength - 2
	if res <= 0 {
		panic(fmt.Errorf("maxItemBytes=%d is insufficient; chunkKeyLength=%d", b.maxItemBytes, chunkKeyLength))
	}
	return res
}

func (b *Big) canStore(key string, valJSON string) bool {
	// Values are stored as strings after converting to JSON.
	// When stored as JSON, Chrome may escape some additional characters.
	// See:
	//   https://chromium.googlesource.com/chromium/chromium/+/707a0ab1f8777bda5aef8aadf6553b4b10f157b2/base/json/string_escape.cc#53
	// Therefore, we give ourselves a significant margin for safety.
	return len(key)+(len(valJSON)*2) <= b.maxItemBytes
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

const (
	// lockResourceID identifies the lock taken to protect against
	// concurrent access during read-modify-write operations. For now, we
	// have a single lock protecting all instances of Big.  We could have
	// different locks for different instances, but the complexity doesn't
	// seem worth it for now.
	lockResourceID = "big-storage-lock"
)

// See PersistentStore.Set().
func (b *Big) Set(ctx jsutil.AsyncContext, data map[string]js.Value) error {
	maxEncodedChunkSize := b.maxChunkSize()
	maxDecodedChunkSize := base64.StdEncoding.DecodedLen(maxEncodedChunkSize)

	chunked := map[string]js.Value{}
	for k, v := range data {
		json := jsutil.ToJSON(v)
		if b.canStore(k, json) {
			// Store directly. Value is small enough.
			chunked[k] = v
			continue
		}

		// Value too large. Break into chunks. There are two caveats:
		// - The JSON string may contain UTF-8 characters, meaning it
		//   we have to be careful about splitting in the middle of a
		//   single character (which occupies 2 bytes).
		// - When stored, Chrome may escape some additional characters
		//   and occupy more space than we compute.  See:
		//     https://chromium.googlesource.com/chromium/chromium/+/707a0ab1f8777bda5aef8aadf6553b4b10f157b2/base/json/string_escape.cc#53
		// To avoid this miscalculations in chunk sizes, we split into
		// chunks such that each chunk, when encoded as base64, fits
		// within the required chunk size.
		manifest := newBigValueManifest()
		for i := 0; i < len(json); i += maxDecodedChunkSize {
			extent := i + maxDecodedChunkSize
			if extent > len(json) {
				extent = len(json)
			}

			// Key is the hash of the contents. This is a simple way
			// to avoid overwriting data.
			chunk := base64.StdEncoding.EncodeToString([]byte(json[i:extent]))
			chunkKey := makeChunkKey(chunk)

			// Add to manifest and data we will store.
			manifest.ChunkKeys = append(manifest.ChunkKeys, chunkKey)
			chunked[chunkKey] = js.ValueOf(chunk)
		}

		// Associate the manifest with the original key.
		chunked[k] = vert.ValueOf(manifest).JSValue()
	}

	var err error
	_, aerr := lock.Async(lockResourceID, func(ctx jsutil.AsyncContext) {
		err = b.s.Set(ctx, chunked)
	}).Await(ctx)
	if aerr != nil {
		return aerr
	}
	return err
}

// See PersistentStore.Get().
func (b *Big) Get(ctx jsutil.AsyncContext) (map[string]js.Value, error) {
	var data map[string]js.Value
	var err error
	_, aerr := lock.Async(lockResourceID, func(ctx jsutil.AsyncContext) {
		data, err = b.s.Get(ctx)
	}).Await(ctx)
	if aerr != nil {
		return nil, aerr
	}
	if err != nil {
		return nil, err
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
					return nil, fmt.Errorf("failed to read data; chunk key %s missing", chunkKey)
				}
				dec, err := base64.StdEncoding.DecodeString(chunkVal.String())
				if err != nil {
					return nil, fmt.Errorf("failed to read data; base64 decode failed: %w", err)
				}

				json.WriteString(string(dec))
			}

			unchunked[k] = jsutil.FromJSON(json.String())
			continue
		}

		// This is just a simple key.
		unchunked[k] = v
	}

	return unchunked, nil
}

// See PersistentStore.Delete().
func (b *Big) Delete(ctx jsutil.AsyncContext, keys []string) error {
	var derr error
	_, aerr := lock.Async(lockResourceID, func(ctx jsutil.AsyncContext) {
		derr = func() error {
			// Delete the requested keys.
			if err := b.s.Delete(ctx, keys); err != nil {
				return err
			}

			// Once successful, delete all chunks that are no longer
			// referenced by any manifest. This takes care of those that
			// were just deleted, as well as any dangling ones that may
			// have been left over from before.
			data, err := b.s.Get(ctx)
			if err != nil {
				return fmt.Errorf("failed to query for dangling chunks: %w", err)
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
			if err := b.s.Delete(ctx, dangling); err != nil {
				return fmt.Errorf("failed to delete dangling chunks: %w", err)
			}
			return nil
		}()
	}).Await(ctx)
	if aerr != nil {
		return aerr
	}
	return derr
}
