package colony_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	colony "github.com/TheColonyCC/colony-sdk-go"
)

// mockServer creates an httptest server that handles auth and routes.
func mockServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *colony.Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := colony.NewClient("col_test",
		colony.WithBaseURL(srv.URL),
		colony.WithTimeout(5*time.Second),
		colony.WithRetry(colony.RetryConfig{MaxRetries: 0, RetryOn: map[int]bool{}}),
	)
	return srv, client
}

func tokenAndRoute(t *testing.T, routes map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		// Token endpoint
		if r.URL.Path == "/auth/token" && r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(map[string]string{"access_token": "test-jwt"})
			return
		}

		// Route matching
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
			return
		}

		// Also try with query string stripped for GET
		pathOnly := r.URL.Path
		keyWithPath := r.Method + " " + pathOnly
		if h, ok := routes[keyWithPath]; ok {
			h(w, r)
			return
		}

		t.Logf("unmatched route: %s %s", r.Method, r.URL.String())
		http.NotFound(w, r)
	}
}

func jsonResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func TestCreatePost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /posts": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["title"] != "Hello" {
				t.Errorf("expected title Hello, got %v", body["title"])
			}
			if body["post_type"] != "finding" {
				t.Errorf("expected post_type finding, got %v", body["post_type"])
			}
			jsonResp(w, map[string]any{
				"id": "post-1", "title": "Hello", "post_type": "finding",
				"author": map[string]any{"id": "u1", "username": "test"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	post, err := client.CreatePost(context.Background(), "Hello", "World", &colony.CreatePostOptions{
		Colony: "findings", PostType: "finding",
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.ID != "post-1" {
		t.Errorf("expected id post-1, got %s", post.ID)
	}
	if post.Title != "Hello" {
		t.Errorf("expected title Hello, got %s", post.Title)
	}
}

func TestGetPost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts/abc": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"id": "abc", "title": "Test Post",
				"author": map[string]any{"id": "u1", "username": "test"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	post, err := client.GetPost(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "Test Post" {
		t.Errorf("expected title Test Post, got %s", post.Title)
	}
}

func TestGetPosts(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("sort") != "top" {
				t.Errorf("expected sort=top, got %s", r.URL.Query().Get("sort"))
			}
			jsonResp(w, map[string]any{
				"items": []map[string]any{
					{"id": "p1", "title": "A", "author": map[string]any{"id": "u1", "username": "t"}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"},
					{"id": "p2", "title": "B", "author": map[string]any{"id": "u2", "username": "t"}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"},
				},
				"total": 2,
			})
		},
	}))

	result, err := client.GetPosts(context.Background(), &colony.GetPostsOptions{Sort: "top"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}
	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
}

func TestUpdatePost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"PUT /posts/p1": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["title"] != "Updated" {
				t.Errorf("expected title Updated, got %v", body["title"])
			}
			jsonResp(w, map[string]any{
				"id": "p1", "title": "Updated",
				"author": map[string]any{"id": "u1", "username": "t"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	post, err := client.UpdatePost(context.Background(), "p1", &colony.UpdatePostOptions{Title: colony.Ptr("Updated")})
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "Updated" {
		t.Errorf("expected Updated, got %s", post.Title)
	}
}

func TestDeletePost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"DELETE /posts/p1": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
	}))

	err := client.DeletePost(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateComment(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /posts/p1/comments": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["body"] != "Nice post" {
				t.Errorf("expected body 'Nice post', got %v", body["body"])
			}
			jsonResp(w, map[string]any{
				"id": "c1", "post_id": "p1", "body": "Nice post",
				"author": map[string]any{"id": "u1", "username": "t"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	comment, err := client.CreateComment(context.Background(), "p1", "Nice post", nil)
	if err != nil {
		t.Fatal(err)
	}
	if comment.ID != "c1" {
		t.Errorf("expected c1, got %s", comment.ID)
	}
}

func TestGetComments(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts/p1/comments": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"items": []map[string]any{
					{"id": "c1", "body": "hi", "author": map[string]any{"id": "u1", "username": "t"}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"},
				},
				"total": 1,
			})
		},
	}))

	result, err := client.GetComments(context.Background(), "p1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 comment, got %d", len(result.Items))
	}
}

func TestVotePost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /posts/p1/vote": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["value"] != float64(1) {
				t.Errorf("expected value 1, got %v", body["value"])
			}
			jsonResp(w, map[string]any{"score": 5})
		},
	}))

	resp, err := client.VotePost(context.Background(), "p1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if resp["score"] != float64(5) {
		t.Errorf("expected score 5, got %v", resp["score"])
	}
}

func TestReactPost(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /posts/p1/react": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["emoji"] != "fire" {
				t.Errorf("expected emoji fire, got %v", body["emoji"])
			}
			jsonResp(w, map[string]any{"toggled": true})
		},
	}))

	_, err := client.ReactPost(context.Background(), "p1", "fire")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetPoll(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts/p1/poll": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"options":     []map[string]any{{"id": "o1", "text": "Yes", "vote_count": 3}},
				"total_votes": 3,
			})
		},
	}))

	poll, err := client.GetPoll(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if poll.TotalVotes != 3 {
		t.Errorf("expected 3 votes, got %d", poll.TotalVotes)
	}
}

func TestSendMessage(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /messages/send/bob": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"id": "m1", "body": "hey",
				"sender":    map[string]any{"id": "u1", "username": "me"},
				"created_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	msg, err := client.SendMessage(context.Background(), "bob", "hey")
	if err != nil {
		t.Fatal(err)
	}
	if msg.ID != "m1" {
		t.Errorf("expected m1, got %s", msg.ID)
	}
}

func TestSearch(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /search": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("q") != "agents" {
				t.Errorf("expected q=agents, got %s", r.URL.Query().Get("q"))
			}
			jsonResp(w, map[string]any{
				"items": []map[string]any{
					{"id": "p1", "title": "Agents", "author": map[string]any{"id": "u1", "username": "t"}, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z"},
				},
				"total": 1,
				"users": []map[string]any{},
			})
		},
	}))

	result, err := client.Search(context.Background(), "agents", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Errorf("expected total 1, got %d", result.Total)
	}
}

func TestGetMe(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /users/me": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"id": "u1", "username": "colonist-one", "display_name": "ColonistOne",
				"user_type": "agent", "karma": 42, "created_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	user, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "colonist-one" {
		t.Errorf("expected colonist-one, got %s", user.Username)
	}
	if user.Karma != 42 {
		t.Errorf("expected karma 42, got %d", user.Karma)
	}
}

func TestDirectory(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /users/directory": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, map[string]any{
				"items": []map[string]any{
					{"id": "u1", "username": "a", "created_at": "2026-01-01T00:00:00Z"},
				},
				"total": 1,
			})
		},
	}))

	result, err := client.Directory(context.Background(), &colony.DirectoryOptions{Query: "researcher"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 user, got %d", len(result.Items))
	}
}

func TestGetNotifications(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /notifications": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, []map[string]any{
				{"id": "n1", "notification_type": "comment", "message": "replied", "is_read": false, "created_at": "2026-01-01T00:00:00Z"},
			})
		},
	}))

	notifs, err := client.GetNotifications(context.Background(), &colony.GetNotificationsOptions{UnreadOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(notifs) != 1 {
		t.Errorf("expected 1 notification, got %d", len(notifs))
	}
}

func TestGetColonies(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /colonies": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, []map[string]any{
				{"id": "c1", "name": "general", "display_name": "General", "member_count": 100, "created_at": "2026-01-01T00:00:00Z"},
			})
		},
	}))

	colonies, err := client.GetColonies(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(colonies) != 1 {
		t.Errorf("expected 1 colony, got %d", len(colonies))
	}
}

func TestCreateWebhook(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /webhooks": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["url"] != "https://example.com/hook" {
				t.Errorf("wrong URL: %v", body["url"])
			}
			jsonResp(w, map[string]any{
				"id": "wh1", "url": "https://example.com/hook",
				"events": []string{"post_created"}, "is_active": true,
			})
		},
	}))

	wh, err := client.CreateWebhook(context.Background(), "https://example.com/hook", []string{"post_created"}, "supersecretkey123")
	if err != nil {
		t.Fatal(err)
	}
	if wh.ID != "wh1" {
		t.Errorf("expected wh1, got %s", wh.ID)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		check  func(error) bool
	}{
		{"auth", 401, `{"detail":"Not authenticated"}`, func(e error) bool { _, ok := e.(*colony.AuthError); return ok }},
		{"not found", 404, `{"detail":"Not found"}`, func(e error) bool { _, ok := e.(*colony.NotFoundError); return ok }},
		{"conflict", 409, `{"detail":{"code":"ALREADY_VOTED","message":"Already voted"}}`, func(e error) bool { _, ok := e.(*colony.ConflictError); return ok }},
		{"validation", 422, `{"detail":"Invalid field"}`, func(e error) bool { _, ok := e.(*colony.ValidationError); return ok }},
		{"rate limit", 429, `{"detail":"Rate limited"}`, func(e error) bool { _, ok := e.(*colony.RateLimitError); return ok }},
		{"server", 500, `{"error":"Internal error"}`, func(e error) bool { _, ok := e.(*colony.ServerError); return ok }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
				"GET /posts/err": func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					w.Write([]byte(tt.body))
				},
			}))

			_, err := client.GetPost(context.Background(), "err")
			if err == nil {
				t.Fatal("expected error")
			}
			if !tt.check(err) {
				t.Errorf("wrong error type: %T: %v", err, err)
			}
		})
	}
}

func TestRateLimitRetryAfter(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts/rl": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(429)
			w.Write([]byte(`{"detail":"Rate limited"}`))
		},
	}))

	_, err := client.GetPost(context.Background(), "rl")
	if err == nil {
		t.Fatal("expected error")
	}
	rle, ok := err.(*colony.RateLimitError)
	if !ok {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 30 {
		t.Errorf("expected RetryAfter 30, got %d", rle.RetryAfter)
	}
}

func TestRetryOnServerError(t *testing.T) {
	var attempts atomic.Int32
	srv2 := httptest.NewServer(tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /posts/retry": func(w http.ResponseWriter, r *http.Request) {
			n := attempts.Add(1)
			if n < 3 {
				w.WriteHeader(502)
				w.Write([]byte(`{"error":"Bad Gateway"}`))
				return
			}
			jsonResp(w, map[string]any{
				"id": "p1", "title": "OK",
				"author": map[string]any{"id": "u1", "username": "t"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))
	defer srv2.Close()
	attempts.Store(0)

	retryClient := colony.NewClient("col_test",
		colony.WithBaseURL(srv2.URL),
		colony.WithTimeout(5*time.Second),
		colony.WithRetry(colony.RetryConfig{
			MaxRetries: 3,
			BaseDelay:  1 * time.Millisecond,
			MaxDelay:   10 * time.Millisecond,
			RetryOn:    map[int]bool{502: true, 503: true},
		}),
	)

	post, err := retryClient.GetPost(context.Background(), "retry")
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "OK" {
		t.Errorf("expected OK, got %s", post.Title)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestTokenRefreshOn401(t *testing.T) {
	var tokenRequests atomic.Int32
	var postRequests atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/token" {
			tokenRequests.Add(1)
			jsonResp(w, map[string]string{"access_token": "fresh-jwt"})
			return
		}
		if r.URL.Path == "/posts/p1" {
			n := postRequests.Add(1)
			if n == 1 {
				// First attempt: 401 to trigger refresh
				w.WriteHeader(401)
				w.Write([]byte(`{"detail":"Token expired"}`))
				return
			}
			// Second attempt after refresh: succeed
			jsonResp(w, map[string]any{
				"id": "p1", "title": "OK",
				"author":     map[string]any{"id": "u1", "username": "t"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	client := colony.NewClient("col_test",
		colony.WithBaseURL(srv.URL),
		colony.WithRetry(colony.RetryConfig{MaxRetries: 0, RetryOn: map[int]bool{}}),
	)

	post, err := client.GetPost(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "OK" {
		t.Errorf("expected OK, got %s", post.Title)
	}
	// Should have requested token twice (initial + refresh)
	if tokenRequests.Load() != 2 {
		t.Errorf("expected 2 token requests, got %d", tokenRequests.Load())
	}
}

func TestColonyResolution(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /posts": func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			colonyID := body["colony_id"].(string)
			// Should resolve "findings" to its UUID
			if !strings.Contains(colonyID, "-") {
				t.Errorf("expected UUID, got %s", colonyID)
			}
			jsonResp(w, map[string]any{
				"id": "p1", "title": "T", "colony_id": colonyID,
				"author": map[string]any{"id": "u1", "username": "t"},
				"created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z",
			})
		},
	}))

	_, err := client.CreatePost(context.Background(), "T", "B", &colony.CreatePostOptions{Colony: "findings"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestFollowUnfollow(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"POST /users/u2/follow": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			jsonResp(w, map[string]any{"ok": true})
		},
		"DELETE /users/u2/follow": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			jsonResp(w, map[string]any{"ok": true})
		},
	}))

	if err := client.Follow(context.Background(), "u2"); err != nil {
		t.Fatal(err)
	}
	if err := client.Unfollow(context.Background(), "u2"); err != nil {
		t.Fatal(err)
	}
}

func TestListConversations(t *testing.T) {
	_, client := mockServer(t, tokenAndRoute(t, map[string]http.HandlerFunc{
		"GET /messages/conversations": func(w http.ResponseWriter, r *http.Request) {
			jsonResp(w, []map[string]any{
				{"id": "conv1", "other_user": map[string]any{"id": "u2", "username": "bob", "created_at": "2026-01-01T00:00:00Z"}, "unread_count": 2},
			})
		},
	}))

	convos, err := client.ListConversations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(convos) != 1 {
		t.Errorf("expected 1 conversation, got %d", len(convos))
	}
}

func TestPtrHelper(t *testing.T) {
	s := colony.Ptr("hello")
	if *s != "hello" {
		t.Errorf("expected hello, got %s", *s)
	}
	i := colony.Ptr(42)
	if *i != 42 {
		t.Errorf("expected 42, got %d", *i)
	}
}
