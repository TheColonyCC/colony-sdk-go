package colony

import (
	"sync"
	"time"
)

// tokenEntry holds a cached JWT token and its expiry.
type tokenEntry struct {
	token  string
	expiry time.Time
}

// globalTokenCache is a process-wide token cache shared across Client
// instances. Keyed by apiKey + "\x00" + baseURL so that clients with the
// same credentials share a token.
var globalTokenCache sync.Map // map[string]*tokenEntry

func tokenCacheKey(apiKey, baseURL string) string {
	return apiKey + "\x00" + baseURL
}

func getCachedToken(apiKey, baseURL string) (string, bool) {
	key := tokenCacheKey(apiKey, baseURL)
	v, ok := globalTokenCache.Load(key)
	if !ok {
		return "", false
	}
	entry := v.(*tokenEntry)
	if time.Now().After(entry.expiry) {
		globalTokenCache.Delete(key)
		return "", false
	}
	return entry.token, true
}

func setCachedToken(apiKey, baseURL, token string) {
	key := tokenCacheKey(apiKey, baseURL)
	globalTokenCache.Store(key, &tokenEntry{
		token:  token,
		expiry: time.Now().Add(tokenCacheDuration),
	})
}

func clearCachedToken(apiKey, baseURL string) {
	key := tokenCacheKey(apiKey, baseURL)
	globalTokenCache.Delete(key)
}
