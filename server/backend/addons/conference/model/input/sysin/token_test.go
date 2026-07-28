package sysin_test

import (
	"context"
	"testing"

	"hotgo/addons/conference/model/input/sysin"
)

func TestTokenCreateInpFilter(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		room    string
		nick    string
		wantErr bool
	}{
		{"ok", "demo", "张三", false},
		{"empty room", "", "张三", true},
		{"empty nick", "demo", "", true},
		{"spaces", "  ", "  ", true},
		{"slash", "a/b", "张三", true},
		{"dotdot", "a..b", "张三", true},
		{"chinese room", "会议室", "张三", true},
		{"ok underscore", "demo_room-1", "lisi", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := &sysin.TokenCreateInp{Room: c.room, Nickname: c.nick}
			err := in.Filter(ctx)
			if c.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
