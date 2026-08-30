package model

import "testing"

func TestIsAdminRole(t *testing.T) {
	tests := []struct {
		role UserRole
		want bool
	}{
		{role: UserRoleAdmin, want: true},
		{role: UserRoleSuperAdmin, want: true},
		{role: UserRoleUser, want: false},
		{role: UserRoleGuest, want: false},
	}
	for _, test := range tests {
		if got := IsAdminRole(test.role); got != test.want {
			t.Fatalf("IsAdminRole(%q) = %v, want %v", test.role, got, test.want)
		}
	}
}
