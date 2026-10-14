package storage

import (
	"errors"
	"reflect"
	"regexp"
	"time"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/users"
	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"go.etcd.io/bbolt"
)

// GetUser allows you to get a user by its id or username. The provided
// id must be a string for username lookup or a uint for id lookup. If id
// is neither, a ErrInvalidDataType will be returned.
func (s *Storage) GetUser(baseScope string, followExternal bool, id any) (*users.User, error) {
	var field string
	switch id.(type) {
	case uint:
		field = "ID"
	case string:
		field = "Username"
	default:
		return nil, fberrors.ErrInvalidDataType
	}

	u, err := s.getBy[users.User](field, id)
	if err != nil {
		return nil, err
	}
	if err := u.Clean(baseScope, followExternal); err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByScope returns the first user whose scope matches the given one, or
// ErrNotExist if none does. The user is returned as stored, without setting up
// its filesystem, as it is meant for existence checks rather than serving.
func (s *Storage) GetUserByScope(scope string) (*users.User, error) {
	u := &users.User{}
	// Match case-insensitively: on a case-insensitive filesystem two scopes
	// that differ only in case (e.g. /users/Alice and /users/alice) resolve to
	// the same home directory, so they must be treated as a collision.
	pattern := "(?i)^" + regexp.QuoteMeta(scope) + "$"
	if err := s.db.Select(q.Re("Scope", pattern)).First(u); err != nil {
		if errors.Is(err, storm.ErrNotFound) {
			return nil, fberrors.ErrNotExist
		}
		return nil, err
	}
	return u, nil
}

func (s *Storage) ListUsers(baseScope string, followExternal bool) ([]*users.User, error) {
	all, err := s.all[users.User]()
	if err != nil {
		return nil, err
	}
	for _, user := range all {
		if err := user.Clean(baseScope, followExternal); err != nil {
			return nil, err
		}
	}
	return all, nil
}

func (s *Storage) SaveUser(user *users.User) error {
	err := user.Clean("", false)
	if err != nil {
		return err
	}
	return s.save(user)
}

func (s *Storage) UpdateUser(user *users.User, fields ...string) error {
	err := user.Clean("", false, fields...)
	if err != nil {
		return err
	}
	if len(fields) == 0 {
		err = s.save(user)
	} else {
		err = s.update(user, fields...)
	}
	if err != nil {
		return err
	}
	s.userMux.Lock()
	defer s.userMux.Unlock()
	s.userUpdated[user.ID] = time.Now().Unix()
	return nil
}

// SaveProvisionedUser saves a user that is being provisioned (via signup, proxy
// auth or hook auth). When its scope was derived from the username, it first
// rejects the save if another user already owns that scope, so that distinct
// usernames cannot silently share one home directory.
//
// The check and the save are held under a single lock. Performing them as two
// independent operations lets two concurrent provisioning requests both observe
// a free scope and both save, leaving two users sharing one home directory.
func (s *Storage) SaveProvisionedUser(user *users.User, derivedScope bool) error {
	if !derivedScope {
		return s.SaveUser(user)
	}
	s.provision.Lock()
	defer s.provision.Unlock()
	_, err := s.GetUserByScope(user.Scope)
	// user should not already exist
	if err == nil {
		return fberrors.ErrExist
	}
	if !errors.Is(err, fberrors.ErrNotExist) {
		return err
	}
	return s.SaveUser(user)
}

// DeleteUser allows you to delete a user by its name or username. The provided
// id must be a string for username lookup or a uint for id lookup. If id
// is neither, a ErrInvalidDataType will be returned.
func (s *Storage) DeleteUser(id any) error {
	user, err := s.GetUser("", false, id)
	if err != nil {
		return err
	}
	if s.isUniqueAdmin(user) {
		return fberrors.ErrRootUserDeletion
	}
	return s.del(user)
}

func (s *Storage) UserLastUpdate(id uint) int64 {
	s.userMux.RLock()
	defer s.userMux.RUnlock()
	return s.userUpdated[id]
}

func (s *Storage) isUniqueAdmin(user *users.User) bool {
	if !user.Perm.Admin {
		return false
	}
	count, err := s.countAdmins()
	if err != nil {
		return true
	}
	return count <= 1
}

func (s *Storage) countAdmins() (int, error) {
	count := 0
	err := s.db.Bolt.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(reflect.TypeFor[users.User]().Name()))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		for _, v := c.First(); v != nil; _, v = c.Next() {
			var u users.User
			if err := s.db.Codec().Unmarshal(v, &u); err != nil {
				return err
			}
			if u.Perm.Admin {
				count++
			}
		}
		return nil
	})
	return count, err
}
