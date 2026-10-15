package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"sync"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/asdine/storm/v3"
)

type Storage struct {
	db *storm.DB

	userMux     sync.RWMutex
	userUpdated map[uint]int64
	provision   sync.Mutex
}

func New(db *storm.DB) *Storage {
	return &Storage{
		db:          db,
		userUpdated: make(map[uint]int64),
	}
}

func Open(path string, perms fs.FileMode) (*Storage, error) {
	db, err := storm.Open(path, storm.BoltOptions(perms, nil))
	if err != nil {
		return nil, err
	}
	s := New(db)
	err = s.configSet("version", 2)
	return s, err
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) getBy[T any](field string, value any) (*T, error) {
	v := new(T)
	err := s.db.One(field, value, v)
	if errors.Is(err, storm.ErrNotFound) {
		return nil, fberrors.ErrNotExist
	}
	return v, err
}

func (s *Storage) all[T any]() ([]*T, error) {
	var all []*T
	err := s.db.All(&all)
	if errors.Is(err, storm.ErrNotFound) {
		return nil, fberrors.ErrNotExist
	}
	return all, err
}

func (s *Storage) update[T any](val *T, fields ...string) error {
	if len(fields) == 0 {
		return s.save(val)
	}

	for _, field := range fields {
		valField := reflect.ValueOf(val).Elem().FieldByName(field)
		if !valField.IsValid() {
			return fmt.Errorf("invalid field: %s", field)
		}
		iface := valField.Interface()
		if err := s.db.UpdateField(val, field, iface); err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) save[T any](val *T) error {
	err := s.db.Save(val)
	if errors.Is(err, storm.ErrAlreadyExists) {
		return fberrors.ErrExist
	}
	return err
}

func (s *Storage) del[T any](val *T) error {
	return s.db.DeleteStruct(val)
}
