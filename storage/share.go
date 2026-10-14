package storage

import (
	"errors"
	"strings"
	"time"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/share"
)

func (s *Storage) AllShares() ([]*share.Link, error) {
	all, err := s.all[share.Link]()
	if err != nil {
		return nil, err
	}
	return all, nil
}

func (s *Storage) SharesForUser(id uint) ([]*share.Link, error) {
	var links []*share.Link
	err := s.db.Select(q.Eq("UserID", id)).Find(&links)
	if errors.Is(err, storm.ErrNotFound) {
		return nil, fberrors.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	return s.dropExpired(links)
}

func (s *Storage) GetShareByHash(hash string) (*share.Link, error) {
	link, err := s.getBy[share.Link]("Hash", hash)
	if err != nil {
		return nil, err
	}
	if link.Expire != 0 && link.Expire <= time.Now().Unix() {
		_ = s.DeleteShare(link.Hash)
		return nil, fberrors.ErrNotExist
	}
	return link, nil
}

func (s *Storage) GetPermanentShare(path string, id uint) (*share.Link, error) {
	link := &share.Link{}
	err := s.db.Select(q.Eq("Path", path), q.Eq("Expire", 0), q.Eq("UserID", id)).First(link)
	if errors.Is(err, storm.ErrNotFound) {
		return nil, fberrors.ErrNotExist
	}
	return link, err
}

func (s *Storage) SharesForPath(path string, id uint) ([]*share.Link, error) {
	var links []*share.Link
	err := s.db.Select(q.Eq("Path", path), q.Eq("UserID", id)).Find(&links)
	if errors.Is(err, storm.ErrNotFound) {
		return nil, fberrors.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	return s.dropExpired(links)
}

func (s *Storage) SaveShare(link *share.Link) error {
	return s.save(link)
}

func (s *Storage) DeleteShare(hash string) error {
	err := s.del(&share.Link{Hash: hash})
	if errors.Is(err, storm.ErrNotFound) {
		return nil
	}
	return err
}

func (s *Storage) DeleteShareWithPathPrefix(pathPrefix string, id uint) error {
	prefix := strings.TrimRight(pathPrefix, "/")
	var links []share.Link
	if err := s.db.Prefix("Path", prefix, &links); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil
		}
		return err
	}
	var err error
	for _, l := range links {
		if l.UserID != id {
			continue
		}
		if l.Path != prefix && !strings.HasPrefix(l.Path, prefix+"/") {
			continue
		}
		err = errors.Join(err, s.db.DeleteStruct(&share.Link{Hash: l.Hash}))
	}
	return err
}

func (s *Storage) dropExpired(links []*share.Link) ([]*share.Link, error) {
	now := time.Now().Unix()
	kept := links[:0]
	for _, l := range links {
		if l.Expire != 0 && l.Expire <= now {
			if err := s.DeleteShare(l.Hash); err != nil {
				return nil, err
			}
			continue
		}
		kept = append(kept, l)
	}
	return kept, nil
}
