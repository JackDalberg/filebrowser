package auth

import (
	"net/http"

	"github.com/JackDalberg/filebrowser/settings"
	"github.com/JackDalberg/filebrowser/users"
)

// Auther is the authentication interface.
type Auther interface {
	// Auth is called to authenticate a request.
	Auth(r *http.Request, usr userStore, stg *settings.Settings, srv *settings.Server) (*users.User, error)
	// LoginPage indicates if this auther needs a login page.
	LoginPage() bool
}

type userStore interface {
	GetUser(baseScope string, followExternalSymlinks bool, id any) (*users.User, error)
	UpdateUser(user *users.User, fields ...string) error
	SaveProvisionedUser(user *users.User, derivedScope bool) error
}
