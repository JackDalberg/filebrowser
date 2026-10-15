package storage

import (
	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/rules"
	"github.com/JackDalberg/filebrowser/settings"
	"github.com/JackDalberg/filebrowser/users"
)

func (s *Storage) GetSettings() (*settings.Settings, error) {
	set := &settings.Settings{}
	if err := s.configGet("settings", set); err != nil {
		return nil, err
	}
	if set.UserHomeBasePath == "" {
		set.UserHomeBasePath = settings.DefaultUsersHomeBasePath
	}
	if set.LogoutPage == "" {
		set.LogoutPage = settings.DefaultLogoutPage
	}
	if set.MinimumPasswordLength == 0 {
		set.MinimumPasswordLength = settings.DefaultMinimumPasswordLength
	}
	if set.Tus.IsZero() {
		set.Tus = settings.Tus{
			ChunkSize:  settings.DefaultTusChunkSize,
			RetryCount: settings.DefaultTusRetryCount,
		}
	}
	if set.FileMode == 0 {
		set.FileMode = settings.DefaultFileMode
	}
	if set.DirMode == 0 {
		set.DirMode = settings.DefaultDirMode
	}
	return set, nil
}

func (s *Storage) SaveSettings(set *settings.Settings) error {
	if len(set.Key) == 0 {
		return fberrors.ErrEmptyKey
	}
	if set.Defaults.Locale == "" {
		set.Defaults.Locale = "en"
	}
	if set.Defaults.ViewMode == "" {
		set.Defaults.ViewMode = users.MosaicViewMode
	}
	if set.Rules == nil {
		set.Rules = []rules.Rule{}
	}
	return s.configSet("settings", set)
}

func (s *Storage) GetServerConfig() (*settings.Server, error) {
	srv := &settings.Server{}
	return srv, s.configGet("server", srv)
}

func (s *Storage) SaveServerConfig(srv *settings.Server) error {
	srv.Clean()
	return s.configSet("server", srv)
}
