package auth

import (
	"net/http"
	"strings"
	"testing"

	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/settings"
	"github.com/JackDalberg/filebrowser/users"
)

type mockUserStore struct {
	users map[string]*users.User
}

func (m *mockUserStore) GetUser(_ string, _ bool, id any) (*users.User, error) {
	switch v := id.(type) {
	case string:
		if u, ok := m.users[v]; ok {
			return u, nil
		}
	case uint:
		for _, u := range m.users {
			if u.ID == v {
				return u, nil
			}
		}
	}
	return nil, fberrors.ErrNotExist
}

func (m *mockUserStore) GetByScope(scope string) (*users.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Scope, scope) {
			return u, nil
		}
	}
	return nil, fberrors.ErrNotExist
}

func (m *mockUserStore) UpdateUser(_ *users.User, _ ...string) error { return nil }

func (m *mockUserStore) SaveProvisionedUser(user *users.User, derivedScope bool) error {
	if derivedScope {
		if _, err := m.GetByScope(user.Scope); err == nil {
			return fberrors.ErrExist
		}
	}
	m.users[user.Username] = user
	return nil
}

func TestProxyAuthCreateUserRestrictsDefaults(t *testing.T) {
	t.Parallel()

	store := &mockUserStore{users: make(map[string]*users.User)}
	srv := &settings.Server{Root: t.TempDir()}

	s := &settings.Settings{
		Key:        []byte("key"),
		AuthMethod: MethodProxyAuth,
		Defaults: settings.UserDefaults{
			Perm: users.Permissions{
				Admin:    true,
				Create:   true,
				Rename:   true,
				Modify:   true,
				Delete:   true,
				Share:    true,
				Download: true,
			},
		},
	}

	auth := ProxyAuth{Header: "X-Remote-User"}
	req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Remote-User", "newproxyuser")

	user, err := auth.Auth(req, store, s, srv)
	if err != nil {
		t.Fatalf("Auth() error: %v", err)
	}

	if user.Perm.Admin {
		t.Error("auto-provisioned proxy user should not have Admin permission")
	}
	if !user.Perm.Create {
		t.Error("auto-provisioned proxy user should retain Create permission from defaults")
	}
}

// With CreateUserDir enabled, two distinct proxy-authenticated users must each
// receive their own home directory instead of both inheriting the server root.
func TestProxyAuthCreateUserDirIsolatesScope(t *testing.T) {
	t.Parallel()

	store := &mockUserStore{users: make(map[string]*users.User)}
	srv := &settings.Server{Root: t.TempDir()}
	s := &settings.Settings{
		Key:              []byte("key"),
		AuthMethod:       MethodProxyAuth,
		CreateUserDir:    true,
		UserHomeBasePath: "/users",
		Defaults: settings.UserDefaults{
			Scope: ".",
			Perm:  users.Permissions{Create: true},
		},
	}

	auth := ProxyAuth{Header: "X-Remote-User"}
	provision := func(name string) *users.User {
		req, _ := http.NewRequest(http.MethodGet, "/", http.NoBody)
		req.Header.Set("X-Remote-User", name)
		u, err := auth.Auth(req, store, s, srv)
		if err != nil {
			t.Fatalf("Auth(%q) error: %v", name, err)
		}
		return u
	}

	alice := provision("alice")
	bob := provision("bob")

	if alice.Scope == "/" || bob.Scope == "/" {
		t.Fatalf("provisioned users inherited the server root: alice=%q bob=%q", alice.Scope, bob.Scope)
	}
	if alice.Scope == bob.Scope {
		t.Fatalf("distinct users must get distinct scopes, both got %q", alice.Scope)
	}
	if alice.Scope != "/users/alice" {
		t.Errorf("expected /users/alice, got %q", alice.Scope)
	}
}
