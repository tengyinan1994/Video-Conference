package sys

import (
	"testing"

	"github.com/livekit/protocol/livekit"
)

func TestRoomHasHumanParticipant(t *testing.T) {
	cases := []struct {
		name string
		in   []*livekit.ParticipantInfo
		want bool
	}{
		{"empty", nil, false},
		{"only egress", []*livekit.ParticipantInfo{
			{Identity: "EG_UCERbEd9wAeh", Name: "EG_recorder"},
		}, false},
		{"egress by name", []*livekit.ParticipantInfo{
			{Identity: "something", Name: "EG_recorder"},
		}, false},
		{"human", []*livekit.ParticipantInfo{
			{Identity: "u_1_ab12cd34", Name: "张三"},
		}, true},
		{"egress then human", []*livekit.ParticipantInfo{
			{Identity: "EG_UCERbEd9wAeh"},
			{Identity: "u_2_deadbeef", Name: "李四"},
		}, true},
		{"nil participant", []*livekit.ParticipantInfo{nil}, false},
	}
	for _, c := range cases {
		if got := roomHasHumanParticipant(c.in); got != c.want {
			t.Fatalf("%s => %v, want %v", c.name, got, c.want)
		}
	}
}
