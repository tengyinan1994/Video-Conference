package sys

import "testing"

func TestParseWindowsInstallerKey(t *testing.T) {
	t.Parallel()

	got, ok := parseWindowsInstallerKey("windows/视频会议_0.1.0_x64-setup.exe")
	if !ok || got.Version != "0.1.0" || got.Key != "windows/视频会议_0.1.0_x64-setup.exe" {
		t.Fatalf("valid key: ok=%v got=%+v", ok, got)
	}

	if _, ok = parseWindowsInstallerKey("/windows/视频会议_1.2.3_x64-setup.exe"); !ok {
		t.Fatal("leading slash should be accepted")
	}

	rejects := []string{
		"",
		"latest.exe",
		"windows/latest.exe",
		"windows/LATEST.EXE",
		"windows/视频会议_latest_x64-setup.exe",
		"windows/视频会议_0.1.0-beta_x64-setup.exe",
		"windows/视频会议_0.1_x64-setup.exe",
		"windows/视频会议_0.1.0.1_x64-setup.exe",
		"windows/视频会议_0.1.0_x64-setup.exe.blockmap",
		"macos/视频会议_0.1.0_x64-setup.exe",
		"视频会议_0.1.0_x64-setup.exe",
	}
	for _, key := range rejects {
		if _, ok = parseWindowsInstallerKey(key); ok {
			t.Fatalf("should reject %q", key)
		}
	}
}

func TestPickLatestWindowsInstaller(t *testing.T) {
	t.Parallel()

	if _, ok := pickLatestWindowsInstaller(nil); ok {
		t.Fatal("empty keys should miss")
	}
	if _, ok := pickLatestWindowsInstaller([]string{"windows/latest.exe", "readme.txt"}); ok {
		t.Fatal("non-matching keys should miss")
	}

	picked, ok := pickLatestWindowsInstaller([]string{
		"windows/latest.exe",
		"windows/视频会议_0.1.9_x64-setup.exe",
		"windows/视频会议_0.10.0_x64-setup.exe",
		"windows/视频会议_0.2.0_x64-setup.exe",
		"windows/视频会议_0.1.10_x64-setup.exe",
	})
	if !ok {
		t.Fatal("want a match")
	}
	if picked.Version != "0.10.0" {
		t.Fatalf("want 0.10.0, got %s (%s)", picked.Version, picked.Key)
	}

	picked, ok = pickLatestWindowsInstaller([]string{
		"windows/视频会议_0.9.9_x64-setup.exe",
		"windows/视频会议_1.0.0_x64-setup.exe",
	})
	if !ok || picked.Version != "1.0.0" {
		t.Fatalf("want 1.0.0, got %+v ok=%v", picked, ok)
	}
}

func TestCompareSemverish(t *testing.T) {
	t.Parallel()
	if compareSemverish("0.10.0", "0.9.0") <= 0 {
		t.Fatal("0.10.0 should be greater than 0.9.0")
	}
	if compareSemverish("0.1.10", "0.1.9") <= 0 {
		t.Fatal("0.1.10 should be greater than 0.1.9")
	}
	if compareSemverish("1.0.0", "1.0.0") != 0 {
		t.Fatal("equal versions")
	}
}
