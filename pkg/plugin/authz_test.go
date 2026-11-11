package plugin

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

const testKID = "test-key"

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func sign(t *testing.T, key any, alg jose.SignatureAlgorithm, c idClaims) string {
	t.Helper()
	opts := (&jose.SignerOptions{}).WithHeader("kid", testKID).WithType("jwt")
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: key}, opts)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := jwt.Signed(signer).Claims(c).Serialize()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func validClaims() idClaims {
	return idClaims{
		Claims: jwt.Claims{
			Subject:  "user:12",
			Audience: jwt.Audience{"org:1"},
			Expiry:   jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
		Username: "viewer",
	}
}

// fakeGrafana answers the endpoints the access check calls. dashboards maps a
// UID to its folder; folders maps a folder to its ancestors.
type fakeGrafana struct {
	key        *ecdsa.PrivateKey
	scopes     []string
	permStatus int
	dashboards map[string]string
	folders    map[string][]string
	folderHits int
}

func (f *fakeGrafana) start(t *testing.T) Settings {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/signing-keys/keys", func(w http.ResponseWriter, _ *http.Request) {
		set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: f.key.Public(), KeyID: testKID, Algorithm: "ES256"}}}
		_ = json.NewEncoder(w).Encode(set)
	})
	mux.HandleFunc("/api/access-control/users/permissions/search", func(w http.ResponseWriter, r *http.Request) {
		if f.permStatus != 0 {
			w.WriteHeader(f.permStatus)
			return
		}
		if r.Header.Get("Authorization") != "Bearer sa-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"12": map[string][]string{"dashboards:read": f.scopes}})
	})
	mux.HandleFunc("/api/dashboards/uid/", func(w http.ResponseWriter, r *http.Request) {
		folder, ok := f.dashboards[strings.TrimPrefix(r.URL.Path, "/api/dashboards/uid/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"dashboard": map[string]any{"title": "T"},
			"meta":      map[string]any{"folderUid": folder},
		})
	})
	mux.HandleFunc("/api/folders/", func(w http.ResponseWriter, r *http.Request) {
		f.folderHits++
		var parents []map[string]string
		for _, uid := range f.folders[strings.TrimPrefix(r.URL.Path, "/api/folders/")] {
			parents = append(parents, map[string]string{"uid": uid})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"parents": parents})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	s := DefaultSettings()
	s.GrafanaURL = srv.URL
	s.GrafanaSAToken = "sa-token"
	return s
}

func TestVerifyIDToken(t *testing.T) {
	key := newKey(t)
	s := (&fakeGrafana{key: key}).start(t)
	service := validClaims()
	service.Subject, service.Username = "service-account:4", ""
	expired := validClaims()
	expired.Expiry = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	otherOrg := validClaims()
	otherOrg.Audience = jwt.Audience{"org:2"}
	otherUser := validClaims()
	otherUser.Username = "admin"
	anonymous := validClaims()
	anonymous.Subject = "anonymous:0"
	noExpiry := validClaims()
	noExpiry.Expiry = nil

	tests := []struct {
		name   string
		raw    string
		login  string
		want   string
		denied bool
	}{
		{name: "user", raw: sign(t, key, jose.ES256, validClaims()), want: "user:12"},
		{name: "service account", raw: sign(t, key, jose.ES256, service), want: "service-account:4"},
		{name: "missing token", raw: "", denied: true},
		{name: "no expiry", raw: sign(t, key, jose.ES256, noExpiry), denied: true},
		{name: "service-account token replayed by a user", raw: sign(t, key, jose.ES256, service), login: "viewer", denied: true},
		{name: "service-account token replayed anonymously", raw: sign(t, key, jose.ES256, service), login: "-", denied: true},
		{name: "service account of another org", raw: sign(t, key, jose.ES256, service), login: "sa-2-reporter", denied: true},
		{name: "garbage", raw: "not-a-token", denied: true},
		{name: "forged with another key", raw: sign(t, newKey(t), jose.ES256, validClaims()), denied: true},
		{name: "symmetric algorithm", raw: sign(t, []byte(strings.Repeat("k", 32)), jose.HS256, validClaims()), denied: true},
		{name: "expired", raw: sign(t, key, jose.ES256, expired), denied: true},
		{name: "other organization", raw: sign(t, key, jose.ES256, otherOrg), denied: true},
		{name: "token of another user", raw: sign(t, key, jose.ES256, otherUser), denied: true},
		{name: "unsupported identity", raw: sign(t, key, jose.ES256, anonymous), denied: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			login := "viewer"
			if tt.want == "service-account:4" {
				login = "sa-1-reporter"
			}
			switch tt.login {
			case "":
			case "-":
				login = ""
			default:
				login = tt.login
			}
			got, err := verifyIDToken(context.Background(), http.DefaultClient, s, tt.raw, login, 1)
			if tt.raw == "" && !errors.Is(err, errNoIdentity) {
				t.Fatalf("a missing token must be reported as such, got %v", err)
			}
			if tt.denied {
				if !errors.Is(err, errAccessDenied) {
					t.Fatalf("want access denied, got %q, %v", got, err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q, %v", got, err)
			}
		})
	}
}

func TestScopesAllow(t *testing.T) {
	folders := []string{"child", "parent"}
	tests := []struct {
		name   string
		scopes []string
		want   bool
	}{
		{"direct grant", []string{"dashboards:uid:d1"}, true},
		{"own folder", []string{"folders:uid:child"}, true},
		{"ancestor folder", []string{"folders:uid:parent"}, true},
		{"all dashboards", []string{"dashboards:*"}, true},
		{"all folders", []string{"folders:*"}, true},
		{"uid wildcards", []string{"folders:uid:*"}, true},
		{"another dashboard", []string{"dashboards:uid:d2"}, false},
		{"another folder", []string{"folders:uid:other"}, false},
		{"a prefix is not a grant", []string{"folders:uid:chil", "dashboards:uid:d"}, false},
		{"empty scope", []string{""}, false},
		{"nothing", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := scopesAllow(tt.scopes, "d1", folders); got != tt.want {
				t.Fatalf("got %v", got)
			}
		})
	}
}

func TestAuthorizeExport(t *testing.T) {
	base := fakeGrafana{
		key:        newKey(t),
		dashboards: map[string]string{"nested": "child", "other": "secret", "root": ""},
		folders:    map[string][]string{"child": {"parent"}},
	}
	tests := []struct {
		name       string
		scopes     []string
		permStatus int
		uids       []string
		denied     bool
		failed     bool
	}{
		{name: "grant on an ancestor reaches a nested dashboard", scopes: []string{"folders:uid:parent"}, uids: []string{"nested"}},
		{name: "root dashboard through the general folder", scopes: []string{"folders:uid:general"}, uids: []string{"root"}},
		{name: "folder outside the grants", scopes: []string{"folders:uid:parent"}, uids: []string{"other"}, denied: true},
		{name: "one forbidden dashboard denies the bundle", scopes: []string{"folders:uid:parent"}, uids: []string{"nested", "other"}, denied: true},
		{name: "unknown dashboard looks like a forbidden one", scopes: []string{"dashboards:*"}, uids: []string{"missing"}, denied: true},
		{name: "no permissions", uids: []string{"nested"}, denied: true},
		{name: "unreadable permissions fail closed", permStatus: http.StatusForbidden, uids: []string{"nested"}, failed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := base
			f.scopes, f.permStatus = tt.scopes, tt.permStatus
			s := f.start(t)
			metas, err := authorizeExport(context.Background(), http.DefaultClient, s, "user:12", tt.uids)
			switch {
			case tt.denied:
				if !errors.Is(err, errAccessDenied) {
					t.Fatalf("want access denied, got %v", err)
				}
			case tt.failed:
				if err == nil || errors.Is(err, errAccessDenied) {
					t.Fatalf("want a failure distinct from a denial, got %v", err)
				}
			default:
				if err != nil || len(metas) != len(tt.uids) {
					t.Fatalf("got %v, %v", metas, err)
				}
			}
			if err != nil && metas != nil {
				t.Fatal("a refusal must not return dashboards")
			}
		})
	}
}

func TestAuthorizeExportLooksUpEachFolderOnce(t *testing.T) {
	f := &fakeGrafana{
		key:        newKey(t),
		scopes:     []string{"folders:uid:parent"},
		dashboards: map[string]string{"a": "child", "b": "child", "c": "child"},
		folders:    map[string][]string{"child": {"parent"}},
	}
	s := f.start(t)
	if _, err := authorizeExport(context.Background(), http.DefaultClient, s, "user:12", []string{"a", "b", "c"}); err != nil {
		t.Fatal(err)
	}
	if f.folderHits != 1 {
		t.Fatalf("%d folder lookups, want 1", f.folderHits)
	}

	f.scopes, f.folderHits = []string{"dashboards:*"}, 0
	if _, err := authorizeExport(context.Background(), http.DefaultClient, s, "user:12", []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if f.folderHits != 0 {
		t.Fatalf("%d folder lookups for a wildcard grant, want 0", f.folderHits)
	}
}
