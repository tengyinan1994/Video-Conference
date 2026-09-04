package sys

import (
	"strings"
	"testing"

	"hotgo/addons/conference/model"
	"hotgo/addons/conference/model/input/sysin"

	"github.com/livekit/protocol/livekit"
)

func TestParseByteRange(t *testing.T) {
	start, end, partial, err := parseByteRange("", 1000)
	if err != nil || partial || start != 0 || end != 999 {
		t.Fatalf("full file: start=%d end=%d partial=%v err=%v", start, end, partial, err)
	}

	start, end, partial, err = parseByteRange("bytes=0-1", 1000)
	if err != nil || !partial || start != 0 || end != 1 {
		t.Fatalf("safari probe: start=%d end=%d partial=%v err=%v", start, end, partial, err)
	}

	start, end, partial, err = parseByteRange("bytes=100-", 1000)
	if err != nil || !partial || start != 100 || end != 999 {
		t.Fatalf("open end: start=%d end=%d partial=%v err=%v", start, end, partial, err)
	}

	start, end, partial, err = parseByteRange("bytes=-50", 1000)
	if err != nil || !partial || start != 950 || end != 999 {
		t.Fatalf("suffix: start=%d end=%d partial=%v err=%v", start, end, partial, err)
	}

	start, end, partial, err = parseByteRange("bytes=0-9999", 1000)
	if err != nil || !partial || start != 0 || end != 999 {
		t.Fatalf("clamp end: start=%d end=%d partial=%v err=%v", start, end, partial, err)
	}

	_, _, _, err = parseByteRange("bytes=1000-1001", 1000)
	if _, ok := err.(*sysin.UnsatisfiableRangeError); !ok {
		t.Fatalf("want unsatisfiable, got %v", err)
	}

	_, _, _, err = parseByteRange("items=0-1", 1000)
	if err == nil {
		t.Fatal("want error for non-bytes unit")
	}
}

func TestApplyPublicEndpoint(t *testing.T) {
	raw := "http://127.0.0.1:17886/recordings/1/1.mp4?X-Amz-Signature=abc"
	got := applyPublicEndpoint(raw, "http://192.168.1.8:17886")
	if !strings.HasPrefix(got, "http://192.168.1.8:17886/") {
		t.Fatalf("host not rewritten: %s", got)
	}
	if !strings.Contains(got, "X-Amz-Signature=abc") {
		t.Fatalf("query lost: %s", got)
	}
	if applyPublicEndpoint(raw, "") != raw {
		t.Fatal("empty public endpoint should keep original")
	}
}

func TestRecordingPresignEndpoint(t *testing.T) {
	cfg := &model.RecordingConfig{
		S3:             model.RecordingS3{Endpoint: "http://rustfs:9000"},
		PublicEndpoint: "http://125.211.217.19:17886",
	}
	if got := recordingPresignEndpoint(cfg); got != "http://125.211.217.19:17886" {
		t.Fatalf("want publicEndpoint, got %s", got)
	}
	cfg.PublicEndpoint = ""
	if got := recordingPresignEndpoint(cfg); got != "http://rustfs:9000" {
		t.Fatalf("want s3.endpoint fallback, got %s", got)
	}
}

func TestRecordingEncodingDefaults(t *testing.T) {
	opt := recordingEncoding(nil)
	if opt.Width != 2560 || opt.Height != 1440 || opt.Framerate != 60 || opt.VideoBitrate != 12000 {
		t.Fatalf("defaults: %+v", opt)
	}
	if opt.VideoCodec != livekit.VideoCodec_H264_HIGH {
		t.Fatalf("codec: %v", opt.VideoCodec)
	}
	opt = recordingEncoding(&model.RecordingConfig{Width: 1920, Height: 1080, Framerate: 30, VideoBitrate: 6000})
	if opt.Width != 1920 || opt.Height != 1080 || opt.Framerate != 30 || opt.VideoBitrate != 6000 {
		t.Fatalf("override: %+v", opt)
	}
}
