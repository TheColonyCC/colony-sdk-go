package colony

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// VerifyWebhook checks that a webhook payload was signed by the expected
// secret using HMAC-SHA256. The signature should come from the
// X-Colony-Signature header.
func VerifyWebhook(payload []byte, signature, secret string) bool {
	// Strip optional "sha256=" prefix.
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// WebhookEvent represents a parsed, verified webhook delivery.
type WebhookEvent struct {
	Event      string         `json:"event"`
	Payload    json.RawMessage `json:"payload"`
	DeliveryID string         `json:"delivery_id,omitempty"`
}

// VerifyAndParseWebhook verifies the signature and parses the payload.
func VerifyAndParseWebhook(payload []byte, signature, secret string) (*WebhookEvent, error) {
	if !VerifyWebhook(payload, signature, secret) {
		return nil, errors.New("colony: webhook signature verification failed")
	}
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
