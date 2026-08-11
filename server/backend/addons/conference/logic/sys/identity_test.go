package sys

import "testing"

func TestMemberIdFromIdentity(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"u_1_ab12cd34", 1},
		{"u_42", 42},
		{"u_99_ff", 99},
		{"g_abcdef", 0},
		{"", 0},
		{"u_x_1", 0},
	}
	for _, c := range cases {
		if got := memberIdFromIdentity(c.in); got != c.want {
			t.Fatalf("%s => %d, want %d", c.in, got, c.want)
		}
	}
}

func TestIsMeetingHostIdentity(t *testing.T) {
	if !isMeetingHostIdentity("u_7_deadbeef", 7) {
		t.Fatal("expected host")
	}
	if isMeetingHostIdentity("u_8_deadbeef", 7) {
		t.Fatal("expected non-host")
	}
	if !isMeetingHostIdentity("u_7", 7) {
		t.Fatal("legacy identity should still match")
	}
}
