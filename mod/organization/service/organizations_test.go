package service

import "testing"

func TestOrganizationStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{"draft", "pending_verification", true},
		{"draft", "closed", true},
		{"pending_verification", "active", true},
		{"pending_verification", "draft", true},
		{"active", "suspended", true},
		{"active", "closed", true},
		{"suspended", "active", true},
		{"suspended", "closed", true},
		{"draft", "active", false},
		{"active", "draft", false},
		{"closed", "active", false},
		{"active", "active", false},
	}

	for _, tc := range cases {
		if got := allowedTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestMemberRoleValidation(t *testing.T) {
	for _, role := range []string{"owner", "manager", "staff", "member"} {
		if !validMemberRole(role) {
			t.Errorf("%q should be a valid member role", role)
		}
	}
	for _, role := range []string{"", "admin", "platform_admin", "OWNER"} {
		if validMemberRole(role) {
			t.Errorf("%q should not be a valid member role", role)
		}
	}
}
