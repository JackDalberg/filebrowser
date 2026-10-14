package storage

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/users"
)

// GetByScope must match case-insensitively: on a case-insensitive filesystem
// two scopes that differ only in case resolve to the same home directory, so
// the provisioning collision check has to treat them as the same scope.
func TestGetByScopeCaseInsensitive(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "db"), fs.ModeTemporary)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SaveUser(&users.User{Username: "CaseVictim", Password: "pw", Scope: "/users/CaseVictim"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	for _, scope := range []string{"/users/CaseVictim", "/users/casevictim", "/USERS/CASEVICTIM"} {
		u, err := st.GetUserByScope(scope)
		if err != nil {
			t.Errorf("GetByScope(%q) unexpected error: %v", scope, err)
			continue
		}
		if u.Username != "CaseVictim" {
			t.Errorf("GetByScope(%q) returned %q, want CaseVictim", scope, u.Username)
		}
	}

	// A genuinely different scope must still miss (and the regex-escaped query
	// must not match a superstring).
	for _, scope := range []string{"/users/other", "/users/CaseVictimX"} {
		if _, err := st.GetUserByScope(scope); !errors.Is(err, fberrors.ErrNotExist) {
			t.Errorf("GetByScope(%q) expected ErrNotExist, got %v", scope, err)
		}
	}
}
