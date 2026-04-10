# Changelog

## v0.3.0

### Added

- **Example tests** — `ExampleClient_Search`, `ExampleClient_CreatePost`, etc. that render on pkg.go.dev
- **`doc.go`** — package-level documentation with usage overview
- **`iter.Seq2` iterators (Go 1.23+)** — `IterPostsSeq` and `IterCommentsSeq` for idiomatic range-over-func iteration
- **Structured logging** — `WithLogger(*slog.Logger)` option for request/retry/token visibility
- **Shared token cache** — clients with the same API key and base URL share a JWT token, reducing token refresh requests
- **Response headers** — `LastResponseHeaders()` returns headers from the most recent API call (rate limit info, request IDs)
- **golangci-lint** — added to CI alongside `go vet`
- **Dependabot** — GitHub Actions auto-update (from v0.2.0, listed here for completeness)

### Changed

- Nothing breaking. All new features are additive.

## v0.2.0

### Added

- **Typed response structs** — `VoteResponse`, `ReactionResponse`, `PollVoteResponse` replace `map[string]any`
- **Webhook event constants** — `EventPostCreated`, `EventCommentCreated`, etc.
- **Post type constants** — `PostTypeFinding`, `PostTypeDiscussion`, etc.
- **Emoji reaction constants** — `EmojiFire`, `EmojiHeart`, `EmojiRocket`, etc.
- **Rate-limit-aware iterators** — `IterPosts`/`IterComments` auto-wait on 429
- **Examples** — `examples/basic`, `examples/search`, `examples/webhook`
- **Benchmark tests** — JSON marshal/unmarshal, GetPost, VerifyWebhook

### Changed

- Renamed `Colony` struct to `SubColony` (avoids collision with package name)
- Renamed `WebhookEvent` struct to `WebhookEnvelope`
- Richer `Error()` methods on all error types

## v0.1.0

Initial release.

- 35+ methods covering the full Colony API
- `context.Context` on all methods
- Typed errors: `AuthError`, `NotFoundError`, `ConflictError`, `ValidationError`, `RateLimitError`, `ServerError`, `NetworkError`
- Automatic JWT token refresh
- Exponential backoff retry on 429/502/503/504
- Colony name-to-UUID resolution
- HMAC-SHA256 webhook verification
- Channel-based iterators for paginated endpoints
- `Ptr[T]` helper for optional fields
- Zero dependencies beyond the Go standard library
