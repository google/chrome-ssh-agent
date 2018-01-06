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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

const (
	validPrivateKey = `
-----BEGIN RSA PRIVATE KEY-----
Proc-Type: 4,ENCRYPTED
DEK-Info: AES-256-CBC,3F17234B07052C56268A529F4C96A478

XeoG0rjwABbZUIKitYRmzYCdLd91HN9zu4d32MkKe+qFNueGp/vJyuFSLYthmw24
RAXxAJ3JE01QGmbeXDrhJR1uuUG/2GtsT2L3jhntxfgMepUOfXuO9YxcO9dd7Qan
P0Zr9Ry59i4IadHYs6ZXpD06nWHHSM/JPxsosfwdrDJTDebo1eVwHVpoizRsL5Wp
2V1a5fBQstGXA55X1fjKlDutrtPsin1picqF5/24A6k1WoeMWhX4UpCkDxj0o+Y6
KMfCipkm/vqKlLHYTLiRruYp7q6zyTKb3Mf+6cVRBwO7DVv1D95nOHkTjymnIsrX
VgdG5z4OcrRjzw9MM4qwTiNv3Ba42veextWWw6spyiE0uYPVbmPFjzLwSCiAXDqY
ckZ9MtD3RoyPjFBz/D4YbtDR7miXR1dzQz55ibk9aZp3PY0JIk3Tc7vPN7hLhmtd
2rgIy4jj8D6SKxWxcMX5suceOcUG7DN1LVPl0K1LNrBe3a9uUsL10W6BsyrqHZF2
X1KiOYI5x2tdlxqSsGUQkmTEIOMPwnb5u/w3d2TvD6p1sgx2z6cRYm1sg/KX1eBy
wz9zQQXZzvo09kh06XSJECWJ+f7nxHj8vh0LREZpHjU16fq2WE0EqMITcHxBDaaE
Aql4BahWuSOEDHGOmMnjMmlBQRSfdDXkHB9WbxD+e4I0guGyQesP347fCIjqhFBJ
RyVjAuvqXNVyqjTBhCxqV5aHRuJFIF7e+drdx0Wn3NWSIelSBxe3zJsARd46xeNL
BqSjKS2Mdetxu2jHLV9RKv0shgeeMzNTl2vXnB6LLeK9grMEHJZfAa8soLtMNb+d
S2M7O9XCX97iwRa0TdxTuLtETOL8JgUnD8wVlJSG7RmqOXLcpA3rbz45kYHDApLR
TtLlRrfTBRLYse12j9i8LxkgefME/nqX5YY8ZUGXCIoifFz/fimj9aIwzIHyvvJG
KlwRQMIg83qTHVqzzgLzfIhmUhFYaB6WaQjW08/9bQUMVh5FnrqHeKAONQMlYbq4
LF0GsvDjBNbN3sbPPDtYP7skj8zhTcfEKznsQkIlFI98fY7tYIsWDG/B5KXEaJ7h
W56PKWnYZ4SWoq2k/FFvnFQT5683bpu5Vl0QJDxUYS8h+6KqGIghmYgR9EGsTbsf
UsJLYJ8Zsh1ZUyOxBvYpqua3tfn3PDIg+UAYf9NI3LkroW7xmkW1CONahGvf1KFn
xvWhPgmazIpHuzb09dlxO6q6OF6yVDPsatJXNnSFTkRKwrRbQR/hVg+oGwjFtWFS
dW5FfsFY8ftVQIozCnxUEw3TwDztRkbVXxQ1O4tOjbzhkkFA3H/NukrbLLCEMjDI
6YoJpONNVsjTVE3Pn+bOc9vuQEpMhITMfvCiI2vqsXMDSYkQNOa/nrvpDGTIEm/w
o038XwjZI5MEPLZlhh69e6jbL3kSsVARD5ahkarLuvUyKpqiInrnUWLHZETwiSH2
wy+03hJeDFxANxHpz25t87FwzHR4FceteqJXHWoR6XiH805u+2KHCHhv+6nvQCe2
SQv684pfXIhZ8Lfr13deSx5G8h+ULUDyfHzgheSXWOPyve+wdehAOyh71npHjyXe
-----END RSA PRIVATE KEY-----`
	validPrivateKeyPassphrase        = "secret"
	validPrivateKeyBlob              = "AAAAB3NzaC1yc2EAAAADAQABAAABAQDvv71z77+977+9G++/ve+/ve+/vSFl77+9Tu+/vSrvv73vv71jFiU+77+9ae+/ve+/ve+/vXfvv71X77+977+9ee+/vciuKzbvv73vv710Eu+/vUpr2aXvv71e77+977+9NMqN77+9SE0V77+977+977+9dUvvv70+77+977+9Se+/vWJ90Ls0VO+/vceOybrvv73vv73vv71v77+9O++/ve+/ve+/vS3Zg++/vTtxa2g0GmEh77+9S++/vVfvv71n77+977+977+9Gdip77+977+977+9Qe+/ve+/vXXvv71lSH7qtq0GTxIM77+977+977+977+9bO+/vcqmaAfvv71HfHLvv71o77+9ZFzvv70wcdmaTu+/ve+/vVDvv71I77+9Hljvv71g77+9R3/vv73vv701fhDvv73vv73vv73vv73vv71A77+977+9Ce+/vX/vv73vv71FEu+/ve+/vV4277+9Ynx077+977+9ICPdsiNN77+977+9QGZnfHzvv73vv73vv702Fu+/vWwEVO+/vWzvv70EUlDvv73vv73vv71CcW7vv71oPERgCxco77+977+9W++/vU0O77+977+9C++/vQhbL++/vWcu77+9"
	validPrivateKeyWithoutPassphrase = `
-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAqvs4Aur/N3tFjDvuAqcLQ4BJVHpoqzO/RbwbXSBA5bCmd7rO
4cy2inJK0oGphOTn6KRxpRJM8Wwl67iZrRYMTgHC357ymzOurMRXN1L1IZNRn4QO
xBoamuX98pMRpPMyBs6dEQoVJAsLaG7ZTdWGXumvGVkGDdit+6waGBqD7XpVl2Q2
iNZHTuhWvSPPoTyhjiv7Nll7zQVpwmwvu/7qdlGES2SH1P4HmZ3E2Qe2KZ1Yj4RE
sig4WoUPLg+xv+qA0gUNZEHxiAKy6Rs787msGS3biUXYhmKUG8aRHxjmxZcqhl62
UnFCzeDQtpovZtlTqtoDZQsZOb3z/1TlxJA8CQIDAQABAoIBAF5FYN6K/uhyOShW
qqYfv+AZzVScoTUztNQYIOY5sE50FXSSNRreKg8vcP2brAGvzAXDFT20V2QNAuNy
xphePa6M3gs5sf3MgxSStJu2S52Vgj13LEUHN4AMKvYiDGpsBDsolAUfEATtaf7M
j1eQ0SNnqLlLEkF0JIlMnJ6JkA/QqrGKbDsIbq0/R2YmF10gS/PAnRnp18vIJ7TX
N201vBGmI9DsTzqTgJ4dnCY8FNxi0Y6dBpxuPpSRE6v5W221Xf56ce60Zi1JAyjq
NrUyrhiVu24SOKwIZ5w1e7ZCLvZjSi3hlezhQgCq2fKl4xf2LpbOdh8hkrXqzSub
yIr5mQECgYEA1v8COltnwnqHip536W4hrjbuiS+w/VznbuU2eAj3zi4DPugib3lX
Xrar8RjTJMPSzV6olIR1D2VyKd3Veiyvi+H+mVPZpfd8OPJRuDU98L26AS0/YAqS
FbGwRZxLPD9x5VfKm5zO4JoSbveCzXhoPpkBulQt9nE4XjKsf/bzhFECgYEAy5c6
JGFQzrnIRcEy87VspOXv1UeMNrNWZGtvwP3R+uOsMK23ySiUOx5FSJRK4IuG20RA
hhIXdWVa7oQfeCbtZSk1yms+3x0T/39p6nLZ2b3nP8WXIsGpjwCk7O52vesAjZgH
i2HFXJ7AWYR8zzxhBj3OKW2l0cBrmtOi+D1H5jkCgYBzUguO49KPFYw4hXHKawFz
4hEm0sb7z+ZvrFEAJ8dL95BUIM2/v3Vm31LxGqC+2q7q67g/GaF0pbSL0mqcgvWS
caFP+xMGm+4s2YWN6jkUNaBc2zlgOatMKahkXkZYxatBGksaFw08mkgC745gyhIY
aZfsqxSQWQCkPkgax4qtUQKBgQCoEWGoIsYowmm4W/OKCN11i3Rf5z6y8X2CTMbm
1SKBMW42iVJNN7iWzTh44CKoF8buP/vcMhc3jMJyYJPyBoC3oDuNrNcsLL8TjsWL
C+EXxZOfq6hGwwUMzoVYKsvPoK7GNRkVUVMyUMONore+BKQ8GM2WmbPn4idymv/Q
WhZ+0QKBgFaSJH2os/hjuyjtHAXOU8ktvudy7IegEUlNUzX0Xzk4eToDvDQNevad
SkxRM2/9n4E6QAADUWlLjVgl92W+lLHylDV5baWe+QKMut3vyXjUJYe1ZKYe6zZV
3wx1s/evfKXpd2Vs4ulNEaVs4nDmZ5zyS7TUp/ByabdkAJ5JnUpR
-----END RSA PRIVATE KEY-----`
)

type storageErrs struct {
	get error
	set error
	del error
}

type memStorage struct {
	data map[string]interface{}
	err  storageErrs
}

func newStorage() *memStorage {
	return &memStorage{
		data: make(map[string]interface{}),
	}
}

func toGopherJSType(v interface{}) interface{} {
	method := reflect.ValueOf(v).MethodByName("Interface")
	if !method.IsValid() {
		return v
	}

	ret := method.Call(nil)
	if len(ret) != 1 {
		return v
	}

	return ret[0].Interface()
}

func (m *memStorage) SetError(err storageErrs) {
	m.err = err
}

func (m *memStorage) Set(data map[string]interface{}, callback func(err error)) {
	if m.err.set != nil {
		callback(m.err.set)
		return
	}

	for k, v := range data {
		m.data[k] = toGopherJSType(v)
	}
	callback(nil)
}

func (m *memStorage) Get(callback func(data map[string]interface{}, err error)) {
	if m.err.get != nil {
		callback(nil, m.err.get)
		return
	}

	// TODO(ralimi) Make a copy.
	callback(m.data, nil)
}

func (m *memStorage) Delete(keys []string, callback func(err error)) {
	if m.err.del != nil {
		callback(m.err.del)
		return
	}

	for _, k := range keys {
		delete(m.data, k)
	}
	callback(nil)
}

func readErr(errc chan error) error {
	for err := range errc {
		return err
	}
	panic("no elements read from channel")
}

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

func findKey(mgr Manager, byId ID, byName string) (ID, error) {
	if byId != InvalidID {
		return byId, nil
	}

	configured, err := syncConfigured(mgr)
	if err != nil {
		return InvalidID, err
	}

	for _, k := range configured {
		if k.Name == byName {
			return k.Id, nil
		}
	}

	return InvalidID, fmt.Errorf("failed to find key with name %s", byName)
}

type initialKey struct {
	Name          string
	PEMPrivateKey string
}

func newTestManager(agent agent.Agent, storage PersistentStore, keys []*initialKey) (Manager, error) {
	mgr := NewManager(agent, storage)
	for _, k := range keys {
		if err := syncAdd(mgr, k.Name, k.PEMPrivateKey); err != nil {
			return nil, err
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
		storageErr     storageErrs
		wantConfigured []string
		wantErr        error
	}{
		{
			description:    "add single key",
			name:           "new-key",
			pemPrivateKey:  validPrivateKey,
			wantConfigured: []string{"new-key"},
		},
		{
			description: "add multiple keys",
			initial: []*initialKey{
				{
					Name:          "new-key-1",
					PEMPrivateKey: validPrivateKey,
				},
			},
			name:           "new-key-2",
			pemPrivateKey:  validPrivateKey,
			wantConfigured: []string{"new-key-1", "new-key-2"},
		},
		{
			description: "add multiple keys with same name",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			name:           "new-key",
			pemPrivateKey:  validPrivateKey,
			wantConfigured: []string{"new-key", "new-key"},
		},
		{
			description:   "reject invalid name",
			name:          "",
			pemPrivateKey: validPrivateKey,
			wantErr:       errors.New("name must not be empty"),
		},
		{
			description:   "fail to write to storage",
			name:          "new-key",
			pemPrivateKey: validPrivateKey,
			storageErr: storageErrs{
				set: errors.New("storage.Set failed"),
			},
			wantErr: errors.New("storage.Set failed"),
		},
	}

	for _, tc := range testcases {
		storage := newStorage()
		mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
		if err != nil {
			t.Fatalf("%s: failed to initialize manager: %v", tc.description, err)
		}

		// Add the key.
		func() {
			storage.SetError(tc.storageErr)
			defer storage.SetError(storageErrs{})

			err := syncAdd(mgr, tc.name, tc.pemPrivateKey)
			if diff := pretty.Diff(err, tc.wantErr); diff != nil {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
		}()

		// Ensure the correct keys are configured at the end.
		configured, err := syncConfigured(mgr)
		if err != nil {
			t.Errorf("%s: failed to get configured keys: %v", tc.description, err)
		}
		names := configuredKeyNames(configured)
		if diff := pretty.Diff(names, tc.wantConfigured); diff != nil {
			t.Errorf("%s: incorrect configured keys; -got +want: %s", tc.description, diff)
		}
	}
}

func TestRemove(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		byName         string
		byId           ID
		storageErr     storageErrs
		wantConfigured []string
		wantErr        error
	}{
		{
			description: "remove single key",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: validPrivateKey,
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
					PEMPrivateKey: validPrivateKey,
				},
			},
			byId:           ID("bogus-id"),
			wantConfigured: []string{"new-key"},
			// Note that it might be nice to return an error here, but
			// the underlying Chrome APIs don't make it trivial to determine
			// if the requested key was removed, or ignored because it didn't
			// exist.  However, it's probably not worth the effort to try
			// and work around this. Therefore, we don't expect an error
			// in this case.
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName: "new-key",
			storageErr: storageErrs{
				get: errors.New("storage.Get failed"),
			},
			wantConfigured: []string{"new-key"},
			wantErr:        errors.New("failed to enumerate keys: failed to read from storage: storage.Get failed"),
		},
		{
			description: "fail to write to storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName: "new-key",
			storageErr: storageErrs{
				del: errors.New("storage.Delete failed"),
			},
			wantConfigured: []string{"new-key"},
			wantErr:        errors.New("failed to delete keys: storage.Delete failed"),
		},
	}

	for _, tc := range testcases {
		storage := newStorage()
		mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
		if err != nil {
			t.Fatalf("%s: failed to initialize manager: %v", tc.description, err)
		}

		// Figure out the ID of the key we will try to remove.
		id, err := findKey(mgr, tc.byId, tc.byName)
		if err != nil {
			t.Fatalf("%s: failed to find key: %v", tc.description, err)
		}

		// Remove the key
		func() {
			storage.SetError(tc.storageErr)
			defer storage.SetError(storageErrs{})

			err := syncRemove(mgr, id)
			if diff := pretty.Diff(err, tc.wantErr); diff != nil {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
		}()

		// Ensure the correct keys are configured at the end.
		configured, err := syncConfigured(mgr)
		if err != nil {
			t.Errorf("%s: failed to get configured keys: %v", tc.description, err)
		}
		names := configuredKeyNames(configured)
		if diff := pretty.Diff(names, tc.wantConfigured); diff != nil {
			t.Errorf("%s: incorrect configured keys; -got +want: %s", tc.description, diff)
		}
	}
}

func TestConfigured(t *testing.T) {
	testcases := []struct {
		description    string
		initial        []*initialKey
		storageErr     storageErrs
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
					PEMPrivateKey: validPrivateKey,
				},
				{
					Name:          "new-key-2",
					PEMPrivateKey: validPrivateKey,
				},
			},
			wantConfigured: []string{"new-key-1", "new-key-2"},
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "new-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			storageErr: storageErrs{
				get: errors.New("storage.Get failed"),
			},
			wantErr: errors.New("failed to read keys: failed to read from storage: storage.Get failed"),
		},
	}

	for _, tc := range testcases {
		storage := newStorage()
		mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
		if err != nil {
			t.Fatalf("%s: failed to initialize manager: %v", tc.description, err)
		}

		// Enumerate the keys.
		func() {
			storage.SetError(tc.storageErr)
			defer storage.SetError(storageErrs{})

			configured, err := syncConfigured(mgr)
			if diff := pretty.Diff(err, tc.wantErr); diff != nil {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
			names := configuredKeyNames(configured)
			if diff := pretty.Diff(names, tc.wantConfigured); diff != nil {
				t.Errorf("%s: incorrect configured keys; -got +want: %s", tc.description, diff)
			}
		}()
	}
}

func TestLoadAndLoaded(t *testing.T) {
	testcases := []struct {
		description string
		initial     []*initialKey
		byName      string
		byId        ID
		passphrase  string
		storageErr  storageErrs
		wantLoaded  []string
		wantErr     error
	}{
		{
			description: "load single key",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName:     "good-key",
			passphrase: validPrivateKeyPassphrase,
			wantLoaded: []string{
				validPrivateKeyBlob,
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
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName:     "good-key",
			passphrase: validPrivateKeyPassphrase,
			wantLoaded: []string{
				validPrivateKeyBlob,
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
			wantErr:    errors.New("failed to parse private key: ssh: no key found"),
		},
		{
			description: "fail on invalid password",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName:     "good-key",
			passphrase: "incorrect passphrase",
			wantErr:    errors.New("failed to parse private key: x509: decryption password incorrect"),
		},
		{
			description: "fail on invalid ID",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byId:       ID("bogus-id"),
			passphrase: "some passphrase",
			wantErr:    errors.New("failed to find key with ID bogus-id"),
		},
		{
			description: "fail to read from storage",
			initial: []*initialKey{
				{
					Name:          "good-key",
					PEMPrivateKey: validPrivateKey,
				},
			},
			byName:     "good-key",
			passphrase: validPrivateKeyPassphrase,
			storageErr: storageErrs{
				get: errors.New("storage.Get failed"),
			},
			wantErr: errors.New("failed to read key: failed to read keys: failed to read from storage: storage.Get failed"),
		},
	}

	for _, tc := range testcases {
		storage := newStorage()
		mgr, err := newTestManager(agent.NewKeyring(), storage, tc.initial)
		if err != nil {
			t.Fatalf("%s: failed to initialize manager: %v", tc.description, err)
		}

		// Figure out the ID of the key we will try to load.
		id, err := findKey(mgr, tc.byId, tc.byName)
		if err != nil {
			t.Fatalf("%s: failed to find key: %v", tc.description, err)
		}

		// Load the key
		func() {
			storage.SetError(tc.storageErr)
			defer storage.SetError(storageErrs{})

			err := syncLoad(mgr, id, tc.passphrase)
			if diff := pretty.Diff(err, tc.wantErr); diff != nil {
				t.Errorf("%s: incorrect error; -got +want: %s", tc.description, diff)
			}
		}()

		// Ensure the correct keys are loaded at the end.
		loaded, err := syncLoaded(mgr)
		if err != nil {
			t.Errorf("%s: failed to get loaded keys: %v", tc.description, err)
		}
		blobs := loadedKeyBlobs(loaded)
		if diff := pretty.Diff(blobs, tc.wantLoaded); diff != nil {
			t.Errorf("%s: incorrect loaded keys; -got +want: %s", tc.description, diff)
		}
	}
}

func TestGetID(t *testing.T) {
	// Create a manager with one configured key.  We load the key and
	// ensure we can correctly extract the ID.
	storage := newStorage()
	agt := agent.NewKeyring()
	mgr, err := newTestManager(agt, storage, []*initialKey{
		{
			Name:          "good-key",
			PEMPrivateKey: validPrivateKey,
		},
	})
	if err != nil {
		t.Fatalf("failed to initialize manager: %v", err)
	}

	// Locate the ID corresponding to the key we configured.
	wantId, err := findKey(mgr, InvalidID, "good-key")
	if err != nil {
		t.Errorf("failed to find ID for good-key: %v", err)
	}

	// Load the key.
	if err := syncLoad(mgr, wantId, validPrivateKeyPassphrase); err != nil {
		t.Errorf("failed to load key: %v", err)
	}

	// Ensure that we can correctly read the ID from the key we loaded.
	loaded, err := syncLoaded(mgr)
	if err != nil {
		t.Errorf("failed to enumerate loaded keys: %v", err)
	}
	if diff := pretty.Diff(loadedKeyIds(loaded), []ID{wantId}); diff != nil {
		t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
	}

	// Now, also load a key into the agent directly (i.e., not through the
	// manager). We will ensure that we get InvalidID back when we try
	// to extract the ID from it.
	priv, err := ssh.ParseRawPrivateKey([]byte(validPrivateKeyWithoutPassphrase))
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
	if diff := pretty.Diff(loadedKeyIds(loaded), []ID{wantId, InvalidID}); diff != nil {
		t.Errorf("incorrect loaded key IDs; -got +want: %s", diff)
	}
}
