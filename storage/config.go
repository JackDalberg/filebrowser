package storage

import (
	"errors"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/asdine/storm/v3"
)

func (s *Storage) configGet(name string, to any) error {
	err := s.db.Get("config", name, to)
	if errors.Is(err, storm.ErrNotFound) {
		return fberrors.ErrNotExist
	}
	return err
}

func (s *Storage) configSet(name string, from any) error {
	return s.db.Set("config", name, from)
}
