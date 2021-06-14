package repo

import (
	"encoding/json"
	"fmt"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
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
