package sysin_test

import (
	"context"
	"strings"
	"testing"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/input/sysin"
)

func TestAdminMeetingTypeEditInpFilter(t *testing.T) {
	ctx := context.Background()

	cases := []struct {
		name     string
		in       *sysin.AdminMeetingTypeEditInp
		wantErr  bool
		wantName string
	}{
		{"normal ok", &sysin.AdminMeetingTypeEditInp{Name: "部门例会"}, false, "部门例会"},
		{"trim spaces ok", &sysin.AdminMeetingTypeEditInp{Name: "  全员大会  "}, false, "全员大会"},
		{"empty reject", &sysin.AdminMeetingTypeEditInp{Name: ""}, true, ""},
		{"blank reject", &sysin.AdminMeetingTypeEditInp{Name: "   "}, true, ""},
		{
			"too long reject",
			&sysin.AdminMeetingTypeEditInp{Name: strings.Repeat("类", consts.MaxMeetingTypeNameLen+1)},
			true,
			"",
		},
		{
			"max length ok",
			&sysin.AdminMeetingTypeEditInp{Name: strings.Repeat("类", consts.MaxMeetingTypeNameLen)},
			false,
			strings.Repeat("类", consts.MaxMeetingTypeNameLen),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.in.Filter(ctx)
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if c.in.Name != c.wantName {
				t.Fatalf("name = %q, want %q", c.in.Name, c.wantName)
			}
		})
	}
}

func TestAdminMeetingTypeViewInpFilter(t *testing.T) {
	ctx := context.Background()

	if err := (&sysin.AdminMeetingTypeViewInp{Id: 1}).Filter(ctx); err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if err := (&sysin.AdminMeetingTypeViewInp{Id: 0}).Filter(ctx); err == nil {
		t.Fatal("want error for id=0, got nil")
	}
}
