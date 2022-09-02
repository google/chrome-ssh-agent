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

	"github.com/google/chrome-ssh-agent/go/jsutil"
	jut "github.com/google/chrome-ssh-agent/go/jsutil/testing"
	"github.com/google/chrome-ssh-agent/go/keys/testdata"
	"github.com/google/chrome-ssh-agent/go/storage"
	st "github.com/google/chrome-ssh-agent/go/storage/testing"
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

func newTestManager(ctx jsutil.AsyncContext, agent agent.Agent, syncStorage, sessionStorage storage.Area, keys []*initialKey) (*DefaultManager, error) {
	mgr := NewManager(agent, syncStorage, sessionStorage)
	for _, k := range keys {
		if err := mgr.Add(ctx, k.Name, k.PEMPrivateKey); err != nil {
			return nil, err
		}

		if k.Load {
			id, err := findKey(ctx, mgr, InvalidID, k.Name)
			if err != nil {
				return nil, err
			}
			if err := mgr.Load(ctx, id, k.Passphrase); err != nil {
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
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {

				syncStorage := storage.NewRaw(st.NewMemArea())
				sessionStorage := storage.NewRaw(st.NewMemArea())
				mgr, err := newTestManager(ctx, agent.NewKeyring(), syncStorage, sessionStorage, tc.initial)
				if err != nil {
					t.Fatalf("failed to initialize manager: %v", err)
				}

				// Add the key.
				err = mgr.Add(ctx, tc.name, tc.pemPrivateKey)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}

				// Ensure the correct keys are configured at the end.
				configured, err := mgr.Configured(ctx)
				if err != nil {
					t.Errorf("failed to get configured keys: %v", err)
				}
				names := configuredKeyNames(configured)
				if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
					t.Errorf("incorrect configured keys; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestRemove(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		byName         string
		byID           ID
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
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				syncStorage := storage.NewRaw(st.NewMemArea())
				sessionStorage := storage.NewRaw(st.NewMemArea())
				mgr, err := newTestManager(ctx, agent.NewKeyring(), syncStorage, sessionStorage, tc.initial)
				if err != nil {
					t.Fatalf("failed to initialize manager: %v", err)
				}

				// Figure out the ID of the key we will try to remove.
				id, err := findKey(ctx, mgr, tc.byID, tc.byName)
				if err != nil {
					t.Fatalf("failed to find key: %v", err)
				}

				// Remove the key
				err = mgr.Remove(ctx, id)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}

				// Ensure the correct keys are configured at the end.
				configured, err := mgr.Configured(ctx)
				if err != nil {
					t.Errorf("failed to get configured keys: %v", err)
				}
				names := configuredKeyNames(configured)
				if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
					t.Errorf("incorrect configured keys; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestConfigured(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
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
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				syncStorage := storage.NewRaw(st.NewMemArea())
				sessionStorage := storage.NewRaw(st.NewMemArea())
				mgr, err := newTestManager(ctx, agent.NewKeyring(), syncStorage, sessionStorage, tc.initial)
				if err != nil {
					t.Fatalf("failed to initialize manager: %v", err)
				}

				// Enumerate the keys.
				configured, err := mgr.Configured(ctx)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}
				names := configuredKeyNames(configured)
				if diff := cmp.Diff(names, tc.wantConfigured); diff != "" {
					t.Errorf("incorrect configured keys; -got +want: %s", diff)
				}
			})
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
			description: "load ecdsa key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.ECDSAWithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.ECDSAWithPassphrase.Passphrase,
			wantLoaded: []string{
				testdata.ECDSAWithPassphrase.Blob,
			},
		},
		{
			description: "load ecdsa key without passphrase",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.ECDSAWithoutPassphrase.Private,
				},
			},
			byName: "good-key",
			wantLoaded: []string{
				testdata.ECDSAWithoutPassphrase.Blob,
			},
		},
		{
			description: "load ed25519 key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.ED25519WithPassphrase.Private,
				},
			},
			byName:     "good-key",
			passphrase: testdata.ED25519WithPassphrase.Passphrase,
			wantLoaded: []string{
				testdata.ED25519WithPassphrase.Blob,
			},
		},
		{
			description: "load ed25519 key without passphrase",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.ED25519WithoutPassphrase.Private,
				},
			},
			byName: "good-key",
			wantLoaded: []string{
				testdata.ED25519WithoutPassphrase.Blob,
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
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				syncStorage := storage.NewRaw(st.NewMemArea())
				sessionStorage := storage.NewRaw(st.NewMemArea())
				mgr, err := newTestManager(ctx, agent.NewKeyring(), syncStorage, sessionStorage, tc.initial)
				if err != nil {
					t.Fatalf("failed to initialize manager: %v", err)
				}

				// Figure out the ID of the key we will try to load.
				id, err := findKey(ctx, mgr, tc.byID, tc.byName)
				if err != nil {
					t.Fatalf("failed to find key: %v", err)
				}

				// Load the key
				err = mgr.Load(ctx, id, tc.passphrase)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}

				// Ensure the correct keys are loaded at the end.
				loaded, err := mgr.Loaded(ctx)
				if err != nil {
					t.Errorf("failed to get loaded keys: %v", err)
				}
				blobs := loadedKeyBlobs(loaded)
				if diff := cmp.Diff(blobs, tc.wantLoaded); diff != "" {
					t.Errorf("incorrect loaded keys; -got +want: %s", diff)
				}

				// Ensure correct keys stored in session
				gotSessionKeys, err := sessionKeyIDs(ctx, mgr.sessionKeys)
				if err != nil {
					t.Errorf("failed to get session keys: %v", err)
				}
				if diff := cmp.Diff(gotSessionKeys, loadedKeyIDs(loaded), idSlice); diff != "" {
					t.Errorf("incorrect session keys; -got +want: %s", diff)
				}
			})
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
		unloadID    ID
		unloadName  string
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
			unloadName: "good-key",
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
			unloadID: ID("bogus-id"),
			wantLoaded: []string{
				testdata.WithPassphrase.Blob,
			},
			wantErr: errAgentUnloadFailed,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.description, func(t *testing.T) {
			jut.DoSync(func(ctx jsutil.AsyncContext) {
				syncStorage := storage.NewRaw(st.NewMemArea())
				sessionStorage := storage.NewRaw(st.NewMemArea())
				mgr, err := newTestManager(ctx, agent.NewKeyring(), syncStorage, sessionStorage, tc.initial)
				if err != nil {
					t.Fatalf("failed to initialize manager: %v", err)
				}

				// Attempt to fill in the ID of the key we will try to unload.
				id := tc.unloadID
				if id == InvalidID {
					id, err = findKey(ctx, mgr, InvalidID, tc.unloadName)
					if err != nil {
						t.Fatalf("failed to get determine key ID to unload: %v", err)
					}
				}

				// Unload the key
				err = mgr.Unload(ctx, id)
				if diff := cmp.Diff(err, tc.wantErr, cmpopts.EquateErrors()); diff != "" {
					t.Errorf("incorrect error; -got +want: %s", diff)
				}

				// Ensure the correct keys are loaded at the end.
				loaded, err := mgr.Loaded(ctx)
				if err != nil {
					t.Errorf("failed to get loaded keys: %v", err)
				}
				blobs := loadedKeyBlobs(loaded)
				if diff := cmp.Diff(blobs, tc.wantLoaded); diff != "" {
					t.Errorf("incorrect loaded keys; -got +want: %s", diff)
				}

				// Ensure correct keys stored in session
				gotSessionKeys, err := sessionKeyIDs(ctx, mgr.sessionKeys)
				if err != nil {
					t.Errorf("failed to get session keys: %v", err)
				}
				if diff := cmp.Diff(gotSessionKeys, loadedKeyIDs(loaded), idSlice); diff != "" {
					t.Errorf("incorrect session keys; -got +want: %s", diff)
				}
			})
		})
	}
}

func TestGetID(t *testing.T) {
	jut.DoSync(func(ctx jsutil.AsyncContext) {
		// Create a manager with one configured key.  We load the key and
		// ensure we can correctly extract the ID.
		syncStorage := storage.NewRaw(st.NewMemArea())
		sessionStorage := storage.NewRaw(st.NewMemArea())
		agt := agent.NewKeyring()
		mgr, err := newTestManager(ctx, agt, syncStorage, sessionStorage, []*initialKey{
			{
				Name:          "good-key",
				PEMPrivateKey: testdata.WithPassphrase.Private,
			},
		})
		if err != nil {
			t.Fatalf("failed to initialize manager: %v", err)
		}

		// Locate the ID corresponding to the key we configured.
		wantID, err := findKey(ctx, mgr, InvalidID, "good-key")
		if err != nil {
			t.Errorf("failed to find ID for good-key: %v", err)
		}

		// Load the key.
		if err = mgr.Load(ctx, wantID, testdata.WithPassphrase.Passphrase); err != nil {
			t.Errorf("failed to load key: %v", err)
		}

		// Ensure that we can correctly read the ID from the key we loaded.
		loaded, err := mgr.Loaded(ctx)
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
		loaded, err = mgr.Loaded(ctx)
		if err != nil {
			t.Errorf("failed to enumerate loaded keys: %v", err)
		}
		if diff := cmp.Diff(loadedKeyIds(loaded), []ID{wantID, InvalidID}); diff != "" {
			t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
		}
	})
}

func TestLoadFromSession(t *testing.T) {
	jut.DoSync(func(ctx jsutil.AsyncContext) {
		// Storage peresists across multiple manager instances
		syncStorage := storage.NewRaw(st.NewMemArea())
		sessionStorage := storage.NewRaw(st.NewMemArea())

		// First manager instance configures and loads a key.
		var wantID ID
		func() {
			agt := agent.NewKeyring()
			mgr, err := newTestManager(ctx, agt, syncStorage, sessionStorage, []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			})
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Locate the ID corresponding to the key we configured.
			wantID, err = findKey(ctx, mgr, InvalidID, "good-key")
			if err != nil {
				t.Errorf("failed to find ID for good-key: %v", err)
			}

			// Load the key.
			if err = mgr.Load(ctx, wantID, testdata.WithPassphrase.Passphrase); err != nil {
				t.Errorf("failed to load key: %v", err)
			}

			// Ensure key is loaded.
			loaded, err := mgr.Loaded(ctx)
			if err != nil {
				t.Errorf("failed to enumerate loaded keys: %v", err)
			}
			if diff := cmp.Diff(loadedKeyIds(loaded), []ID{wantID}); diff != "" {
				t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
			}
		}()

		// Second manager instance loads keys from storage. We expect the
		// loaded key to be loaded into the agent.
		func() {
			agt := agent.NewKeyring()
			mgr, err := newTestManager(ctx, agt, syncStorage, sessionStorage, []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: testdata.WithPassphrase.Private,
				},
			})
			if err != nil {
				t.Fatalf("failed to initialize manager: %v", err)
			}

			// Restore keys from session.
			if err = mgr.LoadFromSession(ctx); err != nil {
				t.Fatalf("failed to load keys from session: %v", err)
			}

			// Ensure key is loaded.
			loaded, err := mgr.Loaded(ctx)
			if err != nil {
				t.Errorf("failed to enumerate loaded keys: %v", err)
			}
			if diff := cmp.Diff(loadedKeyIds(loaded), []ID{wantID}); diff != "" {
				t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
			}
		}()
	})
}
