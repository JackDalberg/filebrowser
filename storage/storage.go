package storage

import (
	"github.com/JackDalberg/filebrowser/auth"
	"github.com/JackDalberg/filebrowser/settings"
	"github.com/JackDalberg/filebrowser/share"
	"github.com/JackDalberg/filebrowser/users"
)

// Storage is a storage powered by a Backend which makes the necessary
// verifications when fetching and saving data to ensure consistency.
type Storage struct {
	Users    users.Store
	Share    *share.Storage
	Auth     *auth.Storage
	Settings *settings.Storage
}
