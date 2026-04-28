package colony

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// --- Pure helpers (no client needed) ---

func TestResolveColonyLegacy(t *testing.T) {
	// Backward-compat helper kept for downstream callers — exercises the
	// known-slug, UUID-shape-passthrough, and unmapped-slug-passthrough
	// branches. New SDK code uses colonyFilterParam / resolveColonyUUID.
	if got := resolveColony("findings"); got != "bbe6be09-da95-4983-b23d-1dd980479a7e" {
		t.Errorf("findings: got %q", got)
	}
	if got := resolveColony("00000000-1111-2222-3333-444455556666"); got != "00000000-1111-2222-3333-444455556666" {
		t.Errorf("uuid passthrough: got %q", got)
	}
	if got := resolveColony("not-in-map"); got != "not-in-map" {
		t.Errorf("unmapped passthrough: got %q", got)
	}
}

func TestColonyFilterParam(t *testing.T) {
	cases := []struct {
		input     string
		wantKey   string
		wantValue string
	}{
		{"findings", "colony_id", "bbe6be09-da95-4983-b23d-1dd980479a7e"},
		{"general", "colony_id", "2e549d01-99f2-459f-8924-48b2690b2170"},
		{"00000000-1111-2222-3333-444455556666", "colony_id", "00000000-1111-2222-3333-444455556666"},
		{"BBE6BE09-DA95-4983-B23D-1DD980479A7E", "colony_id", "BBE6BE09-DA95-4983-B23D-1DD980479A7E"},
		{"builds", "colony", "builds"},
		{"lobby", "colony", "lobby"},
		{"imagining", "colony", "imagining"},
	}
	for _, tc := range cases {
		k, v := colonyFilterParam(tc.input)
		if k != tc.wantKey || v != tc.wantValue {
			t.Errorf("colonyFilterParam(%q) = (%q, %q), want (%q, %q)",
				tc.input, k, v, tc.wantKey, tc.wantValue)
		}
	}
}

// --- resolveColonyUUID (Client method) ---

// slugResolutionTestClient builds a Client whose internal HTTP target is
// httptestServer and whose JWT is pre-seeded so the resolver doesn't need
// to round-trip /auth/token first.
func slugResolutionTestClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" && r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "test-jwt"})
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	key := "col_test_" + t.Name()
	t.Cleanup(func() { clearCachedToken(key, srv.URL) })

	c := NewClient(key, WithBaseURL(srv.URL), WithRetry(RetryConfig{
		MaxRetries: 0,
		RetryOn:    map[int]bool{},
	}))
	return srv, c
}

func TestResolveColonyUUIDFastPath(t *testing.T) {
	// Known slugs and UUID-shaped values short-circuit before any HTTP.
	// We use an httptest.Server that fails on every non-/auth route so any
	// network call would explode the test.
	_, c := slugResolutionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})
	if id, err := c.resolveColonyUUID(context.Background(), "findings"); err != nil || id != "bbe6be09-da95-4983-b23d-1dd980479a7e" {
		t.Errorf("findings: id=%q err=%v", id, err)
	}
	uuid := "11111111-2222-3333-4444-555555555555"
	if id, err := c.resolveColonyUUID(context.Background(), uuid); err != nil || id != uuid {
		t.Errorf("uuid passthrough: id=%q err=%v", id, err)
	}
}

func TestResolveColonyUUIDViaListColonies(t *testing.T) {
	const buildsUUID = "11111111-2222-3333-4444-555555555555"
	var fetches int32
	_, c := slugResolutionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/colonies" {
			atomic.AddInt32(&fetches, 1)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": buildsUUID, "name": "builds"},
				{"id": "99999999-9999-9999-9999-999999999999", "name": "lobby"},
			})
			return
		}
		http.NotFound(w, r)
	})
	id, err := c.resolveColonyUUID(context.Background(), "builds")
	if err != nil || id != buildsUUID {
		t.Fatalf("first call: id=%q err=%v", id, err)
	}
	// Cache reuse — second + third call should NOT trigger another fetch.
	if _, err := c.resolveColonyUUID(context.Background(), "builds"); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if _, err := c.resolveColonyUUID(context.Background(), "lobby"); err != nil {
		t.Fatalf("third call (different cached slug): %v", err)
	}
	if got := atomic.LoadInt32(&fetches); got != 1 {
		t.Errorf("expected 1 GET /colonies, got %d", got)
	}
}

func TestResolveColonyUUIDUnknownSlug(t *testing.T) {
	_, c := slugResolutionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "11111111-2222-3333-4444-555555555555", "name": "builds"},
		})
	})
	_, err := c.resolveColonyUUID(context.Background(), "not-a-real-slug")
	if err == nil {
		t.Fatal("expected error for unmapped slug")
	}
	if !strings.Contains(err.Error(), "not-a-real-slug") {
		t.Errorf("error doesn't mention the slug: %v", err)
	}
	if !strings.Contains(err.Error(), "Check for typos") {
		t.Errorf("error missing the diagnostic hint: %v", err)
	}
}

func TestResolveColonyUUIDListFails(t *testing.T) {
	_, c := slugResolutionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	_, err := c.resolveColonyUUID(context.Background(), "builds")
	if err == nil {
		t.Fatal("expected list-failure to propagate")
	}
	if !strings.Contains(err.Error(), "list colonies failed") {
		t.Errorf("error wrapping lost: %v", err)
	}
}

func TestResolveColonyUUIDSkipsMalformed(t *testing.T) {
	const goodUUID = "11111111-2222-3333-4444-555555555555"
	_, c := slugResolutionTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": goodUUID, "name": "builds"},
			{"id": "", "name": "ghost-no-id"}, // missing id
			{"id": "abc", "name": ""},         // missing name
		})
	})
	id, err := c.resolveColonyUUID(context.Background(), "builds")
	if err != nil || id != goodUUID {
		t.Fatalf("got id=%q err=%v, want goodUUID and no err", id, err)
	}
	// Malformed entries should be skipped, so they're not resolvable.
	if _, err := c.resolveColonyUUID(context.Background(), "ghost-no-id"); err == nil {
		t.Error("expected error for malformed-entry slug")
	}
}
