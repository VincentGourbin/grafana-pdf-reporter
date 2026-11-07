package plugin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

// identityHeader carries the signed identity of the caller. Grafana sets it on
// every resource call made by an authenticated user or service account.
const identityHeader = "X-Grafana-Id"

// errAccessDenied covers every refusal, including an unknown dashboard, so a
// caller cannot use the export endpoints to probe which UIDs exist.
var errAccessDenied = errors.New("access denied")

type idClaims struct {
	jwt.Claims
	Username string `json:"username"`
}

// verifyIDToken returns the typed identity ("user:12", "service-account:4") of
// the caller. Grafana only overwrites the header when it has a token to send,
// so an anonymous request could carry a forged one: the signature is what
// makes the value trustworthy, never its presence.
func verifyIDToken(ctx context.Context, client *http.Client, s Settings, raw, login string, orgID int64) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("%w: no identity token", errAccessDenied)
	}
	parsed, err := jwt.ParseSigned(raw, []jose.SignatureAlgorithm{jose.ES256})
	if err != nil || len(parsed.Headers) == 0 {
		return "", fmt.Errorf("%w: malformed identity token", errAccessDenied)
	}

	var keys jose.JSONWebKeySet
	if err := grafanaGet(ctx, client, s, "/api/signing-keys/keys", "", &keys); err != nil {
		return "", fmt.Errorf("signing keys: %w", err)
	}
	matching := keys.Key(parsed.Headers[0].KeyID)
	if len(matching) == 0 {
		return "", fmt.Errorf("%w: unknown signing key", errAccessDenied)
	}

	var claims idClaims
	if err := parsed.Claims(matching[0].Key, &claims); err != nil {
		return "", fmt.Errorf("%w: invalid identity token signature", errAccessDenied)
	}
	expected := jwt.Expected{
		AnyAudience: jwt.Audience{fmt.Sprintf("org:%d", orgID)},
		Time:        time.Now(),
	}
	if err := claims.Validate(expected); err != nil {
		return "", fmt.Errorf("%w: identity token rejected: %s", errAccessDenied, err.Error())
	}

	switch {
	case strings.HasPrefix(claims.Subject, "user:"):
		if claims.Username != login {
			return "", fmt.Errorf("%w: identity token does not match the caller", errAccessDenied)
		}
	case strings.HasPrefix(claims.Subject, "service-account:"):
	default:
		return "", fmt.Errorf("%w: unsupported identity %q", errAccessDenied, claims.Subject)
	}
	return claims.Subject, nil
}

// fetchReadScopes lists the scopes on which an identity holds dashboards:read.
// The result already folds in basic roles, teams and direct grants.
func fetchReadScopes(ctx context.Context, client *http.Client, s Settings, subject string) ([]string, error) {
	q := url.Values{"namespacedId": {subject}, "action": {"dashboards:read"}}
	var byUser map[string]map[string][]string
	path := "/api/access-control/users/permissions/search?" + q.Encode()
	if err := grafanaGet(ctx, client, s, path, s.GrafanaSAToken, &byUser); err != nil {
		return nil, fmt.Errorf("read permissions of %s: %w", subject, err)
	}
	if len(byUser) > 1 {
		return nil, fmt.Errorf("read permissions of %s: %d identities returned", subject, len(byUser))
	}
	var scopes []string
	for _, actions := range byUser {
		scopes = append(scopes, actions["dashboards:read"]...)
	}
	return scopes, nil
}

// folderChain returns a folder and its ancestors, because a grant on a folder
// applies to everything nested below it. A dashboard at the root has none:
// Grafana names that level "general".
func folderChain(ctx context.Context, client *http.Client, s Settings, folderUID string) ([]string, error) {
	if folderUID == "" {
		return []string{"general"}, nil
	}
	var folder struct {
		Parents []struct {
			UID string `json:"uid"`
		} `json:"parents"`
	}
	if err := grafanaGet(ctx, client, s, "/api/folders/"+url.PathEscape(folderUID), s.GrafanaSAToken, &folder); err != nil {
		return nil, fmt.Errorf("folder %s: %w", folderUID, err)
	}
	chain := []string{folderUID}
	for _, parent := range folder.Parents {
		chain = append(chain, parent.UID)
	}
	return chain, nil
}

// scopesAllow reports whether the scopes cover a dashboard or one of its
// folders. An empty scope grants nothing.
func scopesAllow(scopes []string, uid string, folders []string) bool {
	allowed := map[string]bool{
		"*": true, "dashboards:*": true, "dashboards:uid:*": true,
		"folders:*": true, "folders:uid:*": true,
		"dashboards:uid:" + uid: true,
	}
	for _, folder := range folders {
		allowed["folders:uid:"+folder] = true
	}
	for _, scope := range scopes {
		if allowed[scope] {
			return true
		}
	}
	return false
}

// authorizeExport confirms the caller can read every requested dashboard and
// returns their metadata. It runs in full before the first render, so a bundle
// never produces a partial report for a dashboard the caller cannot open.
func authorizeExport(ctx context.Context, client *http.Client, s Settings, subject string, uids []string) (map[string]DashboardMeta, error) {
	scopes, err := fetchReadScopes(ctx, client, s, subject)
	if err != nil {
		return nil, err
	}
	metas := make(map[string]DashboardMeta, len(uids))
	for _, uid := range uids {
		meta, err := fetchDashboardMeta(ctx, client, s, uid)
		var status *grafanaStatusError
		if errors.As(err, &status) && status.Code == http.StatusNotFound {
			return nil, fmt.Errorf("%w: dashboard %s", errAccessDenied, uid)
		}
		if err != nil {
			return nil, fmt.Errorf("grafana meta %s: %w", uid, err)
		}
		folders, err := folderChain(ctx, client, s, meta.FolderUID)
		if err != nil {
			return nil, err
		}
		if !scopesAllow(scopes, uid, folders) {
			return nil, fmt.Errorf("%w: dashboard %s", errAccessDenied, uid)
		}
		metas[uid] = meta
	}
	return metas, nil
}
