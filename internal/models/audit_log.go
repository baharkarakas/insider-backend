package models

import "time"

type AuditLog struct {
	ID         string                 `json:"id"`
	EntityType string                 `json:"entity_type"`
	EntityID   *string                `json:"entity_id"`
	Action     string                 `json:"action"`
	Details    map[string]any         `json:"details"`
	CreatedAt  time.Time              `json:"created_at"`
}