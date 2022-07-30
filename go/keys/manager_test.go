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

package keys

import (
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

	"github.com/google/chrome-ssh-agent/go/chrome/fakes"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type initialKey struct {
	Name          string
	PEMPrivateKey string
	Load          bool
	Passphrase    string
}

var (
	storageGetErr    = errors.New("Storage.Get() failed")
	storageSetErr    = errors.New("Storage.Set() failed")
	storageDeleteErr = errors.New("Storage.Delete() failed")
)

func newTestManager(agent agent.Agent, storage PersistentStore, keys []*initialKey) (Manager, error) {
	mgr := NewManager(agent, storage)
	for _, k := range keys {
		if err := syncAdd(mgr, k.Name, k.PEMPrivateKey); err != nil {
			return nil, err
		}

		if k.Load {
			id, err := findKey(mgr, InvalidID, k.Name)
			if err != nil {
				return nil, err
			}
			if err := syncLoad(mgr, id, k.Passphrase); err != nil {
				return nil, err
			}
		}
	}

	return mgr, nil
}

func TestAdd(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		name           string
		pemPrivateKey  string
		storageErr     fakes.Errs
		wantConfigured []string
		wantErr        error
	}{
		{
			description:    "add single key",
			name:           "new-key",
			pemPrivateKey:  testdata.WithPassphrase.Private,
			wantConfigured: []string{"new-key"},
		},
		{
			description: "add multiple keys",
			initial: []*initialKey{
				{
					Name:          "new-key-1",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			name:           "new-key-2",
			pemPrivateKey:  testdata.WithPassphrase.Private,
			wantConfigured: []string{"new-key-1", "new-key-2"},
		},
		{
			description: "add multiple keys with same name",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			name:           "new-key",
			pemPrivateKey:  testdata.WithPassphrase.Private,
			wantConfigured: []string{"new-key", "new-key"},
		},
		{
			description:   "reject invalid name",
			name:          "",
			pemPrivateKey: testdata.WithPassphrase.Private,
			wantErr:       errInvalidName,
		},
		{
			description:   "fail to write to storage",
			name:          "new-key",
			pemPrivateKey: testdata.WithPassphrase.Private,
			storageErr: fakes.Errs{
				Set: storageSetErr,
			},
			wantErr: storageSetErr,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			storage := fakes.NewMemStorage()
			mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Add the key.
			func() {
				storage.SetError(tc.storageErr)
				defer storage.SetError(fakes.Errs{})

				ferr := syncAdd(mgr, tc.name, tc.pemPrivateKey)
				if diff := cmp.Diff(ferr, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}
			}()

			// Ensure the correct keys are configured at the end.
			configured, err := syncConfigured(mgr)
			if err != nil {
				t.Errorf("failed to get configured keys: %v", err)
			}
			names := configuredKeyNames(configured)
			if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
				t.Errorf("incorrect configured keys; -got +want: %s", diff)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		byName         string
		byID           ID
		storageErr     fakes.Errs
		wantConfigured []string
		wantErr        error
	}{
		{
			description: "remove single key",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName:         "new-key",
			wantConfigured: nil,
		},
		{
			description: "fail to remove key with invalid ID",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byID:           ID("bogus-id"),
			wantConfigured: []string{"new-key"},
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName: "new-key",
			storageErr: fakes.Errs{
				Get: storageGetErr,
			},
			wantConfigured: []string{"new-key"},
			wantErr:        storageGetErr,
		},
		{
			description: "fail to write to storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName: "new-key",
			storageErr: fakes.Errs{
				Delete: storageDeleteErr,
			},
			wantConfigured: []string{"new-key"},
			wantErr:        storageDeleteErr,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			storage := fakes.NewMemStorage()
			mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Figure out the ID of the key we will try to remove.
			id, err := findKey(mgr, tc.byID, tc.byName)
			if err != nil {
				t.Fatalf("failed to find key: %v", err)
			}

			// Remove the key
			func() {
				storage.SetError(tc.storageErr)
				defer storage.SetError(fakes.Errs{})

				ferr := syncRemove(mgr, id)
				if diff := cmp.Diff(ferr, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}
			}()

			// Ensure the correct keys are configured at the end.
			configured, err := syncConfigured(mgr)
			if err != nil {
				t.Errorf("failed to get configured keys: %v", err)
			}
			names := configuredKeyNames(configured)
			if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
				t.Errorf("incorrect configured keys; -got +want: %s", diff)
			}
		})
	}
}

func TestConfigured(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		storageErr     fakes.Errs
		wantConfigured []string
		wantErr        error
	}{
		{
			description: "empty list on no keys",
		},
		{
			description: "enumerate multiple keys",
			initial: []*initialKey{
				{
					Name:          "new-key-1",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
				{
					Name:          "new-key-2",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			wantConfigured: []string{"new-key-1", "new-key-2"},
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			storageErr: fakes.Errs{
				Get: storageGetErr,
			},
			wantErr: storageGetErr,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			storage := fakes.NewMemStorage()
			mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Enumerate the keys.
			func() {
				storage.SetError(tc.storageErr)
				defer storage.SetError(fakes.Errs{})

				configured, err := syncConfigured(mgr)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}
				names := configuredKeyNames(configured)
				if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
					t.Errorf("incorrect configured keys; -got +want: %s", diff)
				}
			}()
		})
	}
}

func TestLoadAndLoaded(t *testing.T) {
	testcases := []struct {
		description string
		initial     []*initialKey
		byName      string
		byID        ID
		passphrase  string
		storageErr  fakes.Errs
		wantLoaded  []string
		wantErr     error
	}{
		{
			description: "load single key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.WithPassphrase.Passphrase,
			wantLoaded: []string{
				testdata.WithPassphrase.Blob,
			},
		},
		{
			description: "load one of multiple keys",
			initial: []*initialKey{
				{
					Name:          "bad-key",
					PEMPrivateKey: "bogus-key-data",
				},
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.WithPassphrase.Passphrase,
			wantLoaded: []string{
				testdata.WithPassphrase.Blob,
			},
		},
		{
			description: "load unencrypted key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithoutPassphrase.Private,
				},
			},
			byName: "good-key",
			wantLoaded: []string{
				testdata.WithoutPassphrase.Blob,
			},
		},
		{
			description: "load openssh format key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.OpenSSHFormat.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.OpenSSHFormat.Passphrase,
			wantLoaded: []string{
				testdata.OpenSSHFormat.Blob,
			},
		},
		{
			description: "load openssh format key without passphrase",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.OpenSSHFormatWithoutPassphrase.Private,
				},
			},
			byName: "good-key",
			wantLoaded: []string{
				testdata.OpenSSHFormatWithoutPassphrase.Blob,
			},
		},
		{
			description: "load pkcs8 format key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.PKCS8Format.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.PKCS8Format.Passphrase,
			wantLoaded: []string{
				testdata.PKCS8Format.Blob,
			},
		},
		{
			description: "load pkcs8 format key without passphrase",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.PKCS8FormatWithoutPassphrase.Private,
				},
			},
			byName: "good-key",
			wantLoaded: []string{
				testdata.PKCS8FormatWithoutPassphrase.Blob,
			},
		},
		{
			description: "fail on invalid private key",
			initial: []*initialKey{
				{
					Name:          "bad-key",
					PEMPrivateKey: "bogus-key-data",
				},
			},
			byName:     "bad-key",
			passphrase: "some passphrase",
			wantErr:    errParseFailed,
		},
		{
			description: "fail on invalid password",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: "incorrect passphrase",
			wantErr:    x509.IncorrectPasswordError,
		},
		{
			description: "fail on invalid ID",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byID:       ID("bogus-id"),
			passphrase: "some passphrase",
			wantErr:    errKeyNotFound,
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.WithPassphrase.Passphrase,
			storageErr: fakes.Errs{
				Get: storageGetErr,
			},
			wantErr: storageGetErr,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			storage := fakes.NewMemStorage()
			mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Figure out the ID of the key we will try to load.
			id, err := findKey(mgr, tc.byID, tc.byName)
			if err != nil {
				t.Fatalf("failed to find key: %v", err)
			}

			// Load the key
			func() {
				storage.SetError(tc.storageErr)
				defer storage.SetError(fakes.Errs{})

				ferr := syncLoad(mgr, id, tc.passphrase)
				if diff := cmp.Diff(ferr, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}
			}()

			// Ensure the correct keys are loaded at the end.
			loaded, err := syncLoaded(mgr)
			if err != nil {
				t.Errorf("failed to get loaded keys: %v", err)
			}
			blobs := loadedKeyBlobs(loaded)
			if diff := cmp.Diff(blobs, tc.wantLoaded); diff != "" {
				t.Errorf("incorrect loaded keys; -got +want: %s", diff)
			}
		})
	}
}

func makeLoadedKey(format, blob string) *LoadedKey {
	b, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		panic(fmt.Sprintf("failed to decode blob: %v", err))
	}

	result := LoadedKey{Type: format}
	result.SetBlob(b)
	return &result
}

func TestUnload(t *testing.T) {
	testcases := []struct {
		description string
		initial     []*initialKey
		unload      *LoadedKey
		wantLoaded  []string
		wantErr     error
	}{
		{
			description: "unload single key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
					Load:          true,
					Passphrase:    testdata.WithPassphrase.Passphrase,
				},
			},
			unload:     makeLoadedKey(testdata.WithPassphrase.Type, testdata.WithPassphrase.Blob),
			wantLoaded: nil,
		},
		{
			description: "fail on invalid key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
					Load:          true,
					Passphrase:    testdata.WithPassphrase.Passphrase,
				},
			},
			unload: makeLoadedKey("bogus-type", "AAAA"),
			wantLoaded: []string{
				testdata.WithPassphrase.Blob,
			},
			wantErr: errUnloadFailed,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()

			storage := fakes.NewMemStorage()
			mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Unload the key
			err = syncUnload(mgr, tc.unload)
			if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("incorrect error; -got +want: %s", diff)
			}

			// Ensure the correct keys are loaded at the end.
			loaded, err := syncLoaded(mgr)
			if err != nil {
				t.Errorf("failed to get loaded keys: %v", err)
			}
			blobs := loadedKeyBlobs(loaded)
			if diff := cmp.Diff(blobs, tc.wantLoaded); diff != "" {
				t.Errorf("incorrect loaded keys; -got +want: %s", diff)
			}
		})
	}
}

func TestGetID(t *testing.T) {
	// Create a manager with one configured key.  We load the key and
	// ensure we can correctly extract the ID.
	storage := fakes.NewMemStorage()
	agt := agent.NewKeyring()
	mgr, err := newTestManager(agt, storage, []*initialKey{
		{
			Name:          "good-key",
			PEMPrivateKey: testdata.WithPassphrase.Private,
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Locate the ID corresponding to the key we configured.
	wantID, err := findKey(mgr, InvalidID, "good-key")
	if err != nil {
		t.Errorf("failed to find ID for good-key: %v", err)
	}

	// Load the key.
	if err = syncLoad(mgr, wantID, testdata.WithPassphrase.Passphrase); err != nil {
		t.Errorf("failed to load key: %v", err)
	}

	// Ensure that we can correctly read the ID from the key we loaded.
	loaded, err := syncLoaded(mgr)
	if err != nil {
		t.Errorf("failed to enumerate loaded keys: %v", err)
	}
	if diff := cmp.Diff(loadedKeyIds(loaded), []ID{wantID}); diff != "" {
		t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
	}

	// Now, also load a key into the agent directly (i.e., not through the
	// manager). We will ensure that we get InvalidID back when we try
	// to extract the ID from it.
	priv, err := ssh.ParseRawPrivateKey([]byte(testdata.WithoutPassphrase.Private))
	if err != nil {
		t.Errorf("failed to parse private key: %v", err)
	}
	err = agt.Add(agent.AddedKey{
		PrivateKey: priv,
		Comment:    "some comment",
	})
	if err != nil {
		t.Errorf("failed to load key into agent: %v", err)
	}
	loaded, err = syncLoaded(mgr)
	if err != nil {
		t.Errorf("failed to enumerate loaded keys: %v", err)
	}
	if diff := cmp.Diff(loadedKeyIds(loaded), []ID{wantID, InvalidID}); diff != "" {
		t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
	}
}
