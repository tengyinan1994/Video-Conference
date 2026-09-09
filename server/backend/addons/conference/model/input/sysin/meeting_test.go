package sysin_test

import (
	"context"
	"testing"
	"time"

	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/os/gtime"
)

func newStartAt(d time.Duration) *gtime.Time {
	return gtime.Now().Add(d)
}

func TestMeetingCreateInpFilterStartAtNotBeforeNow(t *testing.T) {
	ctx := context.Background()

	start := newStartAt(time.Hour)
	cases := []struct {
		name    string
		startAt *gtime.Time
		wantErr bool
	}{
		{"future start ok", start, false},
		{"future start offset ok", newStartAt(5 * time.Minute), false},
		{"past start reject", newStartAt(-time.Hour), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := &sysin.MeetingCreateInp{
				Title:   "周例会",
				HostName: "张三",
				StartAt: c.startAt,
				EndAt:   c.startAt.Add(time.Hour),
			}
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

func TestMeetingUpdateInpFilterStartAtNotBeforeNow(t *testing.T) {
	ctx := context.Background()

	start := newStartAt(time.Hour)
	cases := []struct {
		name    string
		startAt *gtime.Time
		wantErr bool
	}{
		{"future start ok", start, false},
		{"past start reject", newStartAt(-time.Hour), true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := &sysin.MeetingUpdateInp{
				Id:     1,
				Title:  "周例会",
				StartAt: c.startAt,
				EndAt:  c.startAt.Add(time.Hour),
			}
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

func TestAdminMeetingEditInpFilterStartAtNotBeforeNow(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name    string
		id      int64
		startAt *gtime.Time
		wantErr bool
	}{
		{"create future ok", 0, newStartAt(time.Hour), false},
		{"create past reject", 0, newStartAt(-time.Hour), true},
		{"edit past ok", 1, newStartAt(-time.Hour), false},
		{"edit future ok", 1, newStartAt(time.Hour), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in := &sysin.AdminMeetingEditInp{
				Id:       c.id,
				Title:    "周例会",
				HostName: "张三",
				StartAt:  c.startAt,
				EndAt:    c.startAt.Add(time.Hour),
			}
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
