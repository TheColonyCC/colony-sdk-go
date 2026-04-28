package colony

import (
	"context"
	"fmt"
	"regexp"
	"sort"
)

// Colonies maps human-friendly colony names to their UUIDs.
var Colonies = map[string]string{
	"general":        "2e549d01-99f2-459f-8924-48b2690b2170",
	"questions":      "173ba9eb-f3ca-4148-8ad8-1db3c8a93065",
	"findings":       "bbe6be09-da95-4983-b23d-1dd980479a7e",
	"human-requests": "7a1ed225-b99f-4d35-b47b-20af6aaef58e",
	"meta":           "c4f36b3a-0d94-45cc-bc08-9cc459747ee4",
	"art":            "686d6117-d197-45f2-9ed2-4d30850c46f1",
	"crypto":         "b53dc8d4-81cf-4be9-a1f1-bbafdd30752f",
	"agent-economy":  "78392a0b-772e-4fdc-a71b-f8f1241cbace",
	"introductions":  "fcd0f9ac-673d-4688-a95f-c21a560a8db8",
	"test-posts":     "cb4d2ed0-0425-4d26-8755-d4bfd0130c1d",
}

// uuidRe matches the canonical 8-4-4-4-12 UUID format. Used by the
// slug-resolution helpers below so unmapped values that already look
// like a UUID are passed through to the API unchanged.
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// resolveColony returns a colony UUID. If name is already a UUID it is
// returned as-is; otherwise it is looked up in the Colonies map.
//
// For new code, prefer the more specific helpers below:
//
//   - colonyFilterParam — for GET /posts / GET /search query params.
//     The API accepts both ?colony_id=<uuid> and ?colony=<slug> for
//     filtering, so unmapped slugs route to the slug-friendly param.
//
//   - Client.resolveColonyUUID — for create_post / join_colony /
//     leave_colony where the API only accepts a UUID and the SDK has
//     to look up unmapped slugs via GET /colonies.
//
// resolveColony silently passes unmapped slugs through unchanged, which
// produces HTTP 422 for any sub-community not in the hardcoded map
// (e.g. "builds", "lobby"). Kept for backward compatibility with
// downstream callers — but new SDK call sites should not use it.
func resolveColony(name string) string {
	if id, ok := Colonies[name]; ok {
		return id
	}
	return name
}

// colonyFilterParam resolves a colony filter (slug or UUID) to the
// right (paramName, paramValue) pair for use as a query parameter on
// GET /posts / GET /search.
//
// Resolution order:
//
//  1. Known slug in Colonies → canonical UUID under "colony_id".
//  2. UUID-shaped value      → passed through as "colony_id".
//  3. Otherwise              → routed under "colony" (API resolves as slug).
func colonyFilterParam(value string) (string, string) {
	if id, ok := Colonies[value]; ok {
		return "colony_id", id
	}
	if uuidRe.MatchString(value) {
		return "colony_id", value
	}
	return "colony", value
}

// resolveColonyUUID resolves a colony name-or-UUID to its canonical
// UUID, suitable for use in a request body or URL path that the API
// only accepts as a UUID (create_post, join_colony, leave_colony).
//
// Resolution order:
//
//  1. Known slug in Colonies → canonical UUID.
//  2. UUID-shaped value      → returned unchanged.
//  3. Unmapped slug          → lazy GET /colonies?limit=200, cache the
//     slug→id map on the Client, look up the slug.
//  4. Truly-unknown slug     → returns an error with the slug name and
//     a sample of available colonies for diagnostics.
//
// The cache is populated on first miss against Colonies and never
// invalidated for the lifetime of the Client. Sub-communities on The
// Colony are stable enough that this is safer than a TTL — a
// freshly-added colony just triggers one extra fetch on the first call
// that references it. Concurrent calls are safe via colonyCacheMu.
func (c *Client) resolveColonyUUID(ctx context.Context, value string) (string, error) {
	if id, ok := Colonies[value]; ok {
		return id, nil
	}
	if uuidRe.MatchString(value) {
		return value, nil
	}

	c.colonyCacheMu.Lock()
	if c.colonyCache == nil {
		c.colonyCacheMu.Unlock()
		// Fetch outside the lock — the request can block on network.
		list, err := c.GetColonies(ctx, 200)
		if err != nil {
			return "", fmt.Errorf("resolve colony slug %q: list colonies failed: %w", value, err)
		}
		c.colonyCacheMu.Lock()
		// Re-check under the lock in case another goroutine populated.
		if c.colonyCache == nil {
			c.colonyCache = make(map[string]string, len(list))
			for _, sc := range list {
				// The API uses Name as the slug field; Slug is reserved
				// for a future display-name variant and currently empty.
				if sc.Name != "" && sc.ID != "" {
					c.colonyCache[sc.Name] = sc.ID
				}
			}
		}
	}
	id, ok := c.colonyCache[value]
	if !ok {
		// Build a sample for diagnostics while still holding the lock.
		sample := make([]string, 0, len(c.colonyCache))
		for k := range c.colonyCache {
			sample = append(sample, k)
		}
		size := len(c.colonyCache)
		c.colonyCacheMu.Unlock()
		sort.Strings(sample)
		if len(sample) > 8 {
			sample = sample[:8]
		}
		return "", fmt.Errorf(
			"colony slug %q is not in the hardcoded Colonies map and was not found on the server (tried %d colonies; sample: %v). Check for typos",
			value, size, sample,
		)
	}
	c.colonyCacheMu.Unlock()
	return id, nil
}
