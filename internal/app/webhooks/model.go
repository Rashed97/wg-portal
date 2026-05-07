package webhooks

import (
	"bytes"
	"encoding/json"
	"io"
)

// WebhookData is the data structure for the webhook payload.
type WebhookData struct {
	// Event is the event type (e.g. create, update, delete)
	Event WebhookEvent `json:"event" example:"create"`

	// Entity is the entity type (e.g. user, peer, interface)
	Entity WebhookEntity `json:"entity" example:"user"`

	// Identifier is the identifier of the entity
	Identifier string `json:"identifier" example:"user-123"`

	// Payload is the payload of the event
	Payload any `json:"payload"`
}

// Serialize serializes the WebhookData to JSON and returns it as an io.Reader.
func (d *WebhookData) Serialize() (io.Reader, error) {
	data, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(data), nil
}

type WebhookEntity = string

const (
	WebhookEntityUser       WebhookEntity = "user"
	WebhookEntityPeer       WebhookEntity = "peer"
	WebhookEntityPeerMetric WebhookEntity = "peer_metric"
	WebhookEntityInterface  WebhookEntity = "interface"
)

type WebhookEvent = string

const (
	WebhookEventCreate     WebhookEvent = "create"
	WebhookEventUpdate     WebhookEvent = "update"
	WebhookEventDelete     WebhookEvent = "delete"
	WebhookEventConnect    WebhookEvent = "connect"
	WebhookEventDisconnect WebhookEvent = "disconnect"
	// WebhookEventRegister fires on self-service user registration (first
	// OIDC login auto-creates the user). Distinct from "create" which
	// represents an admin-initiated user record. Wired to TopicUserRegistered
	// in webhooks/manager.go so admin-alert webhooks can act on the
	// "new user just signed up via SSO" event without seeing every
	// admin-driven user CRUD.
	WebhookEventRegister WebhookEvent = "register"
)
