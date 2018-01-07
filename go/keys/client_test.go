package keys

import (
	"errors"
	"reflect"
	"testing"

	"github.com/gopherjs/gopherjs/js"
	"github.com/kr/pretty"
)

type messageHub struct {
	handlers []func(*js.Object, *js.Object, func(interface{})) bool
}

func (m *messageHub) OnMessage(callback func(header *js.Object, sender *js.Object, sendResponse func(interface{})) bool) {
	m.handlers = append(m.handlers, callback)
}

func (m *messageHub) SendMessage(msg interface{}, callback func(rsp *js.Object)) {
	for _, h := range m.handlers {
		h(toJSObject(msg), nil, func(rsp interface{}) {
			callback(toJSObject(rsp))
		})
	}
}

func (m *messageHub) Error() error {
	return nil
}

type dummyManager struct {
	Id             ID
	Name           string
	PEMPrivateKey  string
	Passphrase     string
	ConfiguredKeys []*ConfiguredKey
	LoadedKeys     []*LoadedKey
	Err            error
}

func (m *dummyManager) Configured(callback func(keys []*ConfiguredKey, err error)) {
	callback(m.ConfiguredKeys, m.Err)
}

func (m *dummyManager) Add(name string, pemPrivateKey string, callback func(err error)) {
	m.Name = name
	m.PEMPrivateKey = pemPrivateKey
	callback(m.Err)
}

func (m *dummyManager) Remove(id ID, callback func(err error)) {
	m.Id = id
	callback(m.Err)
}

func (m *dummyManager) Loaded(callback func(keys []*LoadedKey, err error)) {
	callback(m.LoadedKeys, m.Err)
}

func (m *dummyManager) Load(id ID, passphrase string, callback func(err error)) {
	m.Id = id
	m.Passphrase = passphrase
	callback(m.Err)
}

func TestClientServerConfigured(t *testing.T) {
	hub := &messageHub{}
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	k0 := &ConfiguredKey{Object: js.Global.Get("Object").New()}
	k0.Id = ID("id-0")
	k0.Name = "key-0"
	k1 := &ConfiguredKey{Object: js.Global.Get("Object").New()}
	k1.Id = ID("id-1")
	k1.Name = "key-1"

	wantConfiguredKeys := []*ConfiguredKey{k0, k1}
	wantErr := errors.New("failed")

	mgr.ConfiguredKeys = append(mgr.ConfiguredKeys, wantConfiguredKeys...)
	mgr.Err = wantErr

	configured, err := syncConfigured(cli)
	// Compare using reflect.DeepEqual since pretty.Diff fails to
	// terminate on this input.
	if !reflect.DeepEqual(configured, wantConfiguredKeys) {
		t.Errorf("incorrect configured keys; got %s, want %s", configured, wantConfiguredKeys)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerAdd(t *testing.T) {
	hub := &messageHub{}
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantName := "some-name"
	wantPrivateKey := "private-key"
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncAdd(cli, wantName, wantPrivateKey)
	if diff := pretty.Diff(mgr.Name, wantName); diff != nil {
		t.Errorf("incorrect name; -got +want: %s", diff)
	}
	if diff := pretty.Diff(mgr.PEMPrivateKey, wantPrivateKey); diff != nil {
		t.Errorf("incorrect private key; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerRemove(t *testing.T) {
	hub := &messageHub{}
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantId := ID("id-0")
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncRemove(cli, wantId)
	if diff := pretty.Diff(mgr.Id, wantId); diff != nil {
		t.Errorf("incorrect ID; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerLoaded(t *testing.T) {
	hub := &messageHub{}
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	k0 := &LoadedKey{Object: js.Global.Get("Object").New()}
	k0.Type = "type-0"
	k0.Blob = "blob-0"
	k0.Comment = "comment-0"
	k1 := &LoadedKey{Object: js.Global.Get("Object").New()}
	k1.Type = "type-1"
	k1.Blob = "blob-1"
	k1.Comment = "comment-1"

	wantLoadedKeys := []*LoadedKey{k0, k1}
	wantErr := errors.New("failed")

	mgr.LoadedKeys = append(mgr.LoadedKeys, wantLoadedKeys...)
	mgr.Err = wantErr

	loaded, err := syncLoaded(cli)
	// Compare using reflect.DeepEqual since pretty.Diff fails to
	// terminate on this input.
	if !reflect.DeepEqual(loaded, wantLoadedKeys) {
		t.Errorf("incorrect loaded keys; got %s, want %s", loaded, wantLoadedKeys)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}

func TestClientServerLoad(t *testing.T) {
	hub := &messageHub{}
	mgr := &dummyManager{}
	cli := NewClient(hub)
	NewServer(mgr, hub)

	wantId := ID("id-0")
	wantPassphrase := "secret"
	wantErr := errors.New("failed")

	mgr.Err = wantErr

	err := syncLoad(cli, wantId, wantPassphrase)
	if diff := pretty.Diff(mgr.Id, wantId); diff != nil {
		t.Errorf("incorrect ID; -got +want: %s", diff)
	}
	if diff := pretty.Diff(mgr.Passphrase, wantPassphrase); diff != nil {
		t.Errorf("incorrect passphrase; -got +want: %s", diff)
	}
	if diff := pretty.Diff(err, wantErr); diff != nil {
		t.Errorf("incorrect error; -got +want: %s", diff)
	}
}
