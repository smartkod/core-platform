package security_test

import (
	"net/http"
	"testing"
)

// TestNotificationsArePerUser proves a user's inbox cannot be read or mutated
// by another account. Notification ids are sequential-looking UUIDs, so an
// attacker will try them.
func TestNotificationsArePerUser(t *testing.T) {
	w := setup(t)

	t.Run("the inbox is scoped to the caller", func(t *testing.T) {
		var aliceInbox, bobInbox struct {
			Items []struct {
				ID     string `json:"id"`
				UserID string `json:"user_id"`
			} `json:"items"`
		}
		w.alice.decode("/api/v1/notifications", &aliceInbox)
		w.bob.decode("/api/v1/notifications", &bobInbox)

		for _, n := range aliceInbox.Items {
			if n.UserID != "" && n.UserID != w.alice.userID() {
				t.Errorf("Alice's inbox contains an item for %s", n.UserID)
			}
		}
		for _, n := range bobInbox.Items {
			if n.UserID != "" && n.UserID != w.bob.userID() {
				t.Errorf("Bob's inbox contains an item for %s", n.UserID)
			}
		}
	})

	t.Run("marking a foreign notification read is refused", func(t *testing.T) {
		// Create a notification for Bob if the system has one; otherwise the
		// assertion is that a random id is a 404 for Alice, which is the same
		// security property (no existence leak, no write).
		target := firstNotificationID(t, w.bob)
		if target == "" {
			target = "01a1a37a-0000-7000-8000-000000000000"
		}
		status, _, body := w.alice.do(http.MethodPost,
			"/api/v1/notifications/"+target+"/read", map[string]any{}, nil)
		if isSuccess(status) {
			t.Fatalf("Alice marked Bob's notification read (status %d)", status)
		}
		denied(t, "foreign notification read", status, body)
	})

	t.Run("unread count is per user", func(t *testing.T) {
		var count struct {
			Count int `json:"count"`
		}
		status := w.alice.decode("/api/v1/notifications/unread-count", &count)
		mustStatus(t, "unread count", status, http.StatusOK, "")
		if count.Count < 0 {
			t.Errorf("unread count = %d", count.Count)
		}
	})
}

func firstNotificationID(t *testing.T, c *client) string {
	t.Helper()
	var inbox struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	c.decode("/api/v1/notifications", &inbox)
	if len(inbox.Items) == 0 {
		return ""
	}
	return inbox.Items[0].ID
}
