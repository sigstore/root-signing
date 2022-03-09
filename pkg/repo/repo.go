package repo

import (
	"encoding/json"
	"fmt"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"

	"gopkg.in/yaml.v2"
)

func CreateDb(store tuf.LocalStore) (*verify.DB, error) {
	db := verify.NewDB()
	root, err := GetRootFromStore(store)
	if err != nil {
		return nil, err
	}
	for id, k := range root.Keys {
		if err := db.AddKey(id, k); err != nil {
			// ignore ErrWrongID errors by TAP-12
			if _, ok := err.(verify.ErrWrongID); !ok {
				return nil, err
			}
		}
	}
	for name, role := range root.Roles {
		if err := db.AddRole(name, role); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func GetRootFromStore(store tuf.LocalStore) (*data.Root, error) {
	s, err := GetSignedMeta(store, "root.json")
	if err != nil {
		return nil, err
	}
	root := &data.Root{}
	if err := json.Unmarshal(s.Signed, root); err != nil {
		return nil, err
	}
	return root, nil
}

func GetTargetsFromStore(store tuf.LocalStore) (*data.Targets, error) {
	s, err := GetSignedMeta(store, "targets.json")
	if err != nil {
		return nil, err
	}
	t := &data.Targets{}
	if err := json.Unmarshal(s.Signed, t); err != nil {
		return nil, err
	}
	return t, nil
}

func GetSignedMeta(store tuf.LocalStore, name string) (*data.Signed, error) {
	// Get name signed meta (name is of the form *.json)
	meta, err := store.GetMeta()
	if err != nil {
		return nil, err
	}
	signedJSON, ok := meta[name]
	if !ok {
		return nil, fmt.Errorf("missing metadata %s", name)
	}
	s := &data.Signed{}
	if err := json.Unmarshal(signedJSON, s); err != nil {
		return nil, err
	}
	return s, nil
}

func GetMetaFromStore(msg []byte, name string) (interface{}, error) {
	var meta interface{}
	switch name {
	case "root.json":
		meta = &data.Root{}
	case "snapshot.json":
		meta = &data.Snapshot{}
	case "timestamp.json":
		meta = &data.Timestamp{}
	default:
		// Targets and delegated roles
		meta = &data.Targets{}
	}
	if err := json.Unmarshal(msg, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// TODO make public in cosign -- we need to wait for go-tuf changes.
type customMetadata struct {
	Usage  string `json:"usage"`
	Status string `json:"status"`
}

type sigstoreCustomMetadata struct {
	Sigstore customMetadata `json:"sigstore"`
}

// Target metadata helpers
func TargetMetaFromString(b []byte) (map[string]json.RawMessage, error) {
	targetsMeta := map[string]sigstoreCustomMetadata{}
	targetsMetaJSON := map[string]json.RawMessage{}
	if err := yaml.Unmarshal(b, &targetsMeta); err != nil {
		return nil, err
	}
	for t, custom := range targetsMeta {
		customJson, err := json.Marshal(custom)
		if err != nil {
			return nil, err
		}
		targetsMetaJSON[t] = customJson
	}

	return targetsMetaJSON, nil
}
