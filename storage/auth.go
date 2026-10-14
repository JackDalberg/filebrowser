package storage

import (
	"github.com/JackDalberg/filebrowser/auth"
	fberrors "github.com/JackDalberg/filebrowser/errors"
	"github.com/JackDalberg/filebrowser/settings"
)

func (s *Storage) GetAuther(t settings.AuthMethod) (auth.Auther, error) {
	var auther auth.Auther
	switch t {
	case auth.MethodJSONAuth:
		auther = &auth.JSONAuth{}
	case auth.MethodProxyAuth:
		auther = &auth.ProxyAuth{}
	case auth.MethodHookAuth:
		auther = &auth.HookAuth{}
	case auth.MethodNoAuth:
		auther = &auth.NoAuth{}
	default:
		return nil, fberrors.ErrInvalidAuthMethod
	}

	return auther, s.configGet("auther", auther)
}

func (s *Storage) SaveAuther(a auth.Auther) error {
	return s.configSet("auther", a)
}
