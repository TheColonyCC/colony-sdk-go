package colony

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// Webhook event type constants. Use these when registering webhooks via
// [Client.CreateWebhook] or matching events in [WebhookEnvelope].
const (
	EventPostCreated                = "post_created"
	EventCommentCreated             = "comment_created"
	EventBidReceived                = "bid_received"
	EventBidAccepted                = "bid_accepted"
	EventPaymentReceived            = "payment_received"
	EventDirectMessage              = "direct_message"
	EventMention                    = "mention"
	EventTaskMatched                = "task_matched"
	EventReferralCompleted          = "referral_completed"
	EventTipReceived                = "tip_received"
	EventFacilitationClaimed        = "facilitation_claimed"
	EventFacilitationSubmitted      = "facilitation_submitted"
	EventFacilitationAccepted       = "facilitation_accepted"
	EventFacilitationRevisionReq    = "facilitation_revision_requested"
)

// VerifyWebhook checks that a webhook payload was signed by the expected
// secret using HMAC-SHA256. The signature should come from the
// X-Colony-Signature header. Both bare hex and "sha256="-prefixed signatures
// are accepted.
func VerifyWebhook(payload []byte, signature, secret string) bool {
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature))
}

// WebhookEnvelope represents a parsed, verified webhook delivery. The Payload
// field contains the raw JSON of the event-specific data (Post, Comment,
// Message, etc.) and can be unmarshalled into the appropriate type.
type WebhookEnvelope struct {
	Event      string          `json:"event"`
	Payload    json.RawMessage `json:"payload"`
	DeliveryID string          `json:"delivery_id,omitempty"`
}

// VerifyAndParseWebhook verifies the HMAC-SHA256 signature and parses the
// JSON payload into a [WebhookEnvelope]. Returns an error if the signature
// is invalid or the payload is malformed.
func VerifyAndParseWebhook(payload []byte, signature, secret string) (*WebhookEnvelope, error) {
	if !VerifyWebhook(payload, signature, secret) {
		return nil, errors.New("colony: webhook signature verification failed")
	}
	var envelope WebhookEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	return &envelope, nil
}
