package core

import "testing"

func TestVaryTokenPolicy(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct{ input, want string }{
		{"", "X-Inertia"},
		{"Accept-Encoding", "Accept-Encoding, X-Inertia"},
		{"Accept-Encoding, x-INERTIA ", "Accept-Encoding, x-INERTIA "},
		{"*", "*"},
		{"Accept-Encoding, *", "Accept-Encoding, *"},
	} {
		if got := VaryValue(fixture.input, HeaderInertia); got != fixture.want {
			t.Errorf("VaryValue(%q): got %q, want %q", fixture.input, got, fixture.want)
		}
	}
}
