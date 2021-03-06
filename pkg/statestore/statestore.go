// Package statestore provides an abstraction on top of go-statestore that allows
// you to make a StateStore that tracks its own version and knows how to
// migrate itself to the target version
package statestore

import (
	"context"

	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/namespace"
	cbg "github.com/whyrusleeping/cbor-gen"

	"github.com/chenjianmei111/go-statestore"

	"github.com/chenjianmei111/go-ds-versioning/internal/migrate"
	"github.com/chenjianmei111/go-ds-versioning/internal/runner"
	"github.com/chenjianmei111/go-ds-versioning/internal/utils"
	versioning "github.com/chenjianmei111/go-ds-versioning/pkg"
)

// StoredState is an interface for accessing state
type StoredState interface {
	End() error
	Get(out cbg.CBORUnmarshaler) error
	Mutate(mutator interface{}) error
}

// StateStore is a wrapper around a datastore for saving CBOR encoded structs
type StateStore interface {
	Begin(i interface{}, state interface{}) error
	Get(i interface{}) StoredState
	Has(i interface{}) (bool, error)
	List(out interface{}) error
}

type migratedStateStore struct {
	ss *statestore.StateStore
	ms versioning.MigrationState
}

// NewVersionedStateStore sets takes a datastore, fsm parameters, migrations list, and target version, and returns
// an fsm whose functions will fail till it's migrated to the target version and a function to run migrations
func NewVersionedStateStore(ds datastore.Batching, migrations versioning.VersionedMigrationList, target versioning.VersionKey) (StateStore, func(context.Context) error) {
	r := runner.NewRunner(ds, migrations, target, migrate.To)
	ss := statestore.New(namespace.Wrap(ds, datastore.NewKey(string(target))))
	return NewMigratedStateStore(ss, r), r.Migrate
}

// NewMigratedStateStore returns an fsm whose functions will fail until the migration state says its ready
func NewMigratedStateStore(ss *statestore.StateStore, ms versioning.MigrationState) StateStore {
	return &migratedStateStore{ss, ms}
}

func (mss *migratedStateStore) Begin(i interface{}, state interface{}) error {
	if err := mss.ms.ReadyError(); err != nil {
		return err
	}
	return mss.ss.Begin(i, state)
}

func (mss *migratedStateStore) Get(i interface{}) StoredState {
	if err := mss.ms.ReadyError(); err != nil {
		return &utils.NotReadyStoredState{Err: err}
	}
	return mss.ss.Get(i)
}

func (mss *migratedStateStore) Has(i interface{}) (bool, error) {
	if err := mss.ms.ReadyError(); err != nil {
		return false, err
	}
	return mss.ss.Has(i)
}

func (mss *migratedStateStore) List(out interface{}) error {
	if err := mss.ms.ReadyError(); err != nil {
		return err
	}
	return mss.ss.List(out)
}
