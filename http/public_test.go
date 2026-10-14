package fbhttp

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/JackDalberg/filebrowser/rules"
	"github.com/JackDalberg/filebrowser/settings"
	"github.com/JackDalberg/filebrowser/share"
	"github.com/JackDalberg/filebrowser/storage"
	"github.com/JackDalberg/filebrowser/users"
)

func TestPublicShareHandlerAuthentication(t *testing.T) {
	t.Parallel()

	const passwordBcrypt = "$2y$10$TFAmdCbyd/mEZDe5fUeZJu.MaJQXRTwdqb/IQV.eTn6dWrF58gCSe"
	testCases := map[string]struct {
		share              *share.Link
		req                *http.Request
		sharePerm          bool
		downloadPerm       bool
		expectedStatusCode int
	}{
		"Public share, no auth required": {
			share:              &share.Link{Hash: "h", UserID: 1},
			req:                newHTTPRequest(t),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 200,
		},
		"Private share, no auth provided, 401": {
			share:              &share.Link{Hash: "h", UserID: 1, PasswordHash: passwordBcrypt, Token: "123"},
			req:                newHTTPRequest(t),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 401,
		},
		"Private share, authentication via token": {
			share:              &share.Link{Hash: "h", UserID: 1, PasswordHash: passwordBcrypt, Token: "123"},
			req:                newHTTPRequest(t, func(r *http.Request) { r.URL.RawQuery = "token=123" }),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 200,
		},
		"Private share, authentication via invalid token, 401": {
			share:              &share.Link{Hash: "h", UserID: 1, PasswordHash: passwordBcrypt, Token: "123"},
			req:                newHTTPRequest(t, func(r *http.Request) { r.URL.RawQuery = "token=1234" }),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 401,
		},
		"Private share, authentication via password": {
			share:              &share.Link{Hash: "h", UserID: 1, PasswordHash: passwordBcrypt, Token: "123"},
			req:                newHTTPRequest(t, func(r *http.Request) { r.Header.Set("X-SHARE-PASSWORD", "password") }),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 200,
		},
		"Private share, authentication via invalid password, 401": {
			share:              &share.Link{Hash: "h", UserID: 1, PasswordHash: passwordBcrypt, Token: "123"},
			req:                newHTTPRequest(t, func(r *http.Request) { r.Header.Set("X-SHARE-PASSWORD", "wrong-password") }),
			sharePerm:          true,
			downloadPerm:       true,
			expectedStatusCode: 401,
		},
		"Share owner lost share permission, 403": {
			share:              &share.Link{Hash: "h", UserID: 1},
			req:                newHTTPRequest(t),
			sharePerm:          false,
			downloadPerm:       true,
			expectedStatusCode: 403,
		},
		"Share owner lost download permission, 403": {
			share:              &share.Link{Hash: "h", UserID: 1},
			req:                newHTTPRequest(t),
			sharePerm:          true,
			downloadPerm:       false,
			expectedStatusCode: 403,
		},
	}

	for name, tc := range testCases {
		for handlerName, handler := range map[string]handleFunc{"public share handler": publicShareHandler, "public dl handler": publicDlHandler} {
			t.Run(fmt.Sprintf("%s: %s", handlerName, name), func(t *testing.T) {
				t.Parallel()

				st, err := storage.Open(filepath.Join(t.TempDir(), "db"), fs.ModeTemporary)
				if err != nil {
					t.Fatalf("failed to open db: %v", err)
				}
				t.Cleanup(func() { _ = st.Close() })

				scope := t.TempDir()

				if err := st.SaveShare(tc.share); err != nil {
					t.Fatalf("failed to save share: %v", err)
				}
				if err := st.SaveUser(&users.User{
					Username: "username",
					Password: "pw",
					Scope:    "/",
					Perm: users.Permissions{
						Share:    tc.sharePerm,
						Download: tc.downloadPerm,
					},
				}); err != nil {
					t.Fatalf("failed to save user: %v", err)
				}
				if err := st.SaveSettings(&settings.Settings{Key: []byte("key")}); err != nil {
					t.Fatalf("failed to save settings: %v", err)
				}

				recorder := httptest.NewRecorder()
				handler := handle(handler, "", st, &settings.Server{Root: scope})

				handler.ServeHTTP(recorder, tc.req)
				result := recorder.Result()
				defer result.Body.Close()
				if result.StatusCode != tc.expectedStatusCode {
					t.Errorf("expected status code %d, got status code %d", tc.expectedStatusCode, result.StatusCode)
				}
			})
		}
	}
}

// TestPublicShareHandlerRules ensures that owner rules keep applying to paths
// below a shared directory, even though the share rebases the filesystem onto
// that directory. A deny rule relative to the owner's scope must not be
// bypassable by requesting the blocked path through the public share.
func TestPublicShareHandlerRules(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		handler            handleFunc
		path               string
		expectedStatusCode int
	}{
		"blocked file via dl handler, 403": {
			handler:            publicDlHandler,
			path:               "h/private/secret.txt",
			expectedStatusCode: 403,
		},
		"blocked dir listing via share handler, 403": {
			handler:            publicShareHandler,
			path:               "h/private/",
			expectedStatusCode: 403,
		},
		"blocked dir download via dl handler, 403": {
			handler:            publicDlHandler,
			path:               "h/private/",
			expectedStatusCode: 403,
		},
		"allowed file via dl handler, 200": {
			handler:            publicDlHandler,
			path:               "h/public/readme.txt",
			expectedStatusCode: 200,
		},
		"allowed dir listing via share handler, 200": {
			handler:            publicShareHandler,
			path:               "h/public/",
			expectedStatusCode: 200,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			st, err := storage.Open(filepath.Join(t.TempDir(), "db"), fs.ModeTemporary)
			if err != nil {
				t.Fatalf("failed to open db: %v", err)
			}
			t.Cleanup(func() { _ = st.Close() })

			scope := t.TempDir()
			for _, dir := range []string{"projects/private", "projects/public"} {
				if err := os.MkdirAll(filepath.Join(scope, dir), 0o755); err != nil {
					t.Fatalf("failed to create dir: %v", err)
				}
			}
			if err := os.WriteFile(filepath.Join(scope, "projects/private/secret.txt"), []byte("top secret"), 0o600); err != nil {
				t.Fatalf("failed to write secret file: %v", err)
			}
			if err := os.WriteFile(filepath.Join(scope, "projects/public/readme.txt"), []byte("hello"), 0o600); err != nil {
				t.Fatalf("failed to write public file: %v", err)
			}

			if err := st.SaveShare(&share.Link{Hash: "h", UserID: 1, Path: "/projects"}); err != nil {
				t.Fatalf("failed to save share: %v", err)
			}
			if err := st.SaveUser(&users.User{
				Username: "username",
				Password: "pw",
				Scope:    "/",
				Perm:     users.Permissions{Share: true, Download: true},
				Rules: []rules.Rule{
					{Allow: false, Path: "/projects/private"},
				},
			}); err != nil {
				t.Fatalf("failed to save user: %v", err)
			}
			if err := st.SaveSettings(&settings.Settings{Key: []byte("key")}); err != nil {
				t.Fatalf("failed to save settings: %v", err)
			}

			req := newHTTPRequest(t, func(r *http.Request) { r.URL.Path = tc.path })

			recorder := httptest.NewRecorder()
			handler := handle(tc.handler, "", st, &settings.Server{Root: scope})

			handler.ServeHTTP(recorder, req)
			result := recorder.Result()
			defer result.Body.Close()
			if result.StatusCode != tc.expectedStatusCode {
				t.Errorf("expected status code %d, got status code %d", tc.expectedStatusCode, result.StatusCode)
			}
		})
	}
}

func newHTTPRequest(t *testing.T, requestModifiers ...func(*http.Request)) *http.Request {
	t.Helper()
	r, err := http.NewRequest(http.MethodGet, "h", http.NoBody)
	if err != nil {
		t.Fatalf("failed to construct request: %v", err)
	}
	for _, modify := range requestModifiers {
		modify(r)
	}
	return r
}
