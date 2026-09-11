package types

import "testing"

func TestUserDisplayName(t *testing.T) {
	tests := []struct {
		name string
		user User
		want string
	}{
		{
			name: "nickname",
			user: User{Nickname: "  Profile name  ", Firstname: "First", Lastname: "Last", Username: "account"},
			want: "Profile name",
		},
		{
			name: "full name",
			user: User{Firstname: "  First", Lastname: "Last  ", Username: "account"},
			want: "First Last",
		},
		{
			name: "username",
			user: User{Username: "  account  "},
			want: "account",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.user.DisplayName(); got != test.want {
				t.Fatalf("DisplayName() = %q, want %q", got, test.want)
			}
		})
	}
}
