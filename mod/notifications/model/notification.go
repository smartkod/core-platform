// Package model holds the notifications persistence types.
package model

import (
	"encoding/json"
	"time"
)

// Notification is one queued message.
type Notification struct {
	ID            string     `gorm:"column:id;primaryKey"`
	UserID        string     `gorm:"column:user_id"`
	Channel       string     `gorm:"column:channel"`
	Template      string     `gorm:"column:template"`
	Subject       string     `gorm:"column:subject"`
	Body          string     `gorm:"column:body"`
	Payload       string     `gorm:"column:payload"`
	Status        string     `gorm:"column:status"`
	Attempts      int        `gorm:"column:attempts"`
	LastError     string     `gorm:"column:last_error"`
	NextAttemptAt time.Time  `gorm:"column:next_attempt_at"`
	ReadAt        *time.Time `gorm:"column:read_at"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	SentAt        *time.Time `gorm:"column:sent_at"`
}

// TableName pins the table so a struct rename cannot silently change the schema.
func (Notification) TableName() string { return "notifications" }

// PayloadMap decodes the JSON payload, tolerating an empty value.
func (n *Notification) PayloadMap() map[string]string {
	out := map[string]string{}
	if n.Payload == "" {
		return out
	}
	_ = json.Unmarshal([]byte(n.Payload), &out)
	return out
}

// Preference is one user's channel opt-in state.
type Preference struct {
	UserID    string    `gorm:"column:user_id;primaryKey"`
	Channels  string    `gorm:"column:channels"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName pins the table name.
func (Preference) TableName() string { return "notification_preferences" }

// ChannelsMap decodes the JSON channel map.
func (p *Preference) ChannelsMap() map[string]bool {
	out := map[string]bool{}
	if p.Channels == "" {
		return out
	}
	_ = json.Unmarshal([]byte(p.Channels), &out)
	return out
}
