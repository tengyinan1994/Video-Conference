package sys

import (
	"context"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"

	"hotgo/addons/conference/service"
	"hotgo/internal/library/contexts"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/minio/minio-go/v7"
)

const (
	clientReleaseBucket    = "releases"
	windowsReleasePrefix   = "windows/"
	windowsReleasePlatform = "windows"
)

// windows/视频会议_<semver>_x64-setup.exe，version 仅允许数字段，禁止 latest.exe
var windowsInstallerKeyRe = regexp.MustCompile(`^windows/视频会议_(\d+\.\d+\.\d+)_x64-setup\.exe$`)

type sSysClientRelease struct{}

type windowsInstaller struct {
	Key     string
	Version string
}

func NewSysClientRelease() *sSysClientRelease {
	return &sSysClientRelease{}
}

func init() {
	service.RegisterSysClientRelease(NewSysClientRelease())
}

func (s *sSysClientRelease) OpenForDownload(ctx context.Context, platform string) (rc io.ReadCloser, filename string, size int64, err error) {
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, "", 0, gerror.New("请先登录")
	}
	if strings.ToLower(strings.TrimSpace(platform)) != windowsReleasePlatform {
		return nil, "", 0, gerror.New("该平台暂无可用安装包")
	}

	cfg, err := loadRecordingConfig(ctx)
	if err != nil {
		return nil, "", 0, err
	}
	if strings.TrimSpace(cfg.S3.Endpoint) == "" || strings.TrimSpace(cfg.S3.AccessKey) == "" || strings.TrimSpace(cfg.S3.SecretKey) == "" {
		return nil, "", 0, gerror.New("对象存储未配置")
	}

	client, err := newRecordingS3Client(cfg)
	if err != nil {
		return nil, "", 0, err
	}

	var keys []string
	for obj := range client.ListObjects(ctx, clientReleaseBucket, minio.ListObjectsOptions{
		Prefix:    windowsReleasePrefix,
		Recursive: true,
	}) {
		if obj.Err != nil {
			return nil, "", 0, gerror.Wrap(obj.Err, "查询安装包失败")
		}
		keys = append(keys, obj.Key)
	}

	picked, ok := pickLatestWindowsInstaller(keys)
	if !ok {
		return nil, "", 0, gerror.New("暂无可用的 Windows 安装包")
	}

	obj, err := client.GetObject(ctx, clientReleaseBucket, picked.Key, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, gerror.Wrap(err, "读取安装包失败")
	}
	stat, statErr := obj.Stat()
	if statErr != nil {
		_ = obj.Close()
		return nil, "", 0, gerror.Wrap(statErr, "读取安装包失败")
	}
	return obj, path.Base(picked.Key), stat.Size, nil
}

func parseWindowsInstallerKey(key string) (windowsInstaller, bool) {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" || strings.HasSuffix(strings.ToLower(key), "latest.exe") {
		return windowsInstaller{}, false
	}
	m := windowsInstallerKeyRe.FindStringSubmatch(key)
	if len(m) != 2 {
		return windowsInstaller{}, false
	}
	return windowsInstaller{Key: key, Version: m[1]}, true
}

func pickLatestWindowsInstaller(keys []string) (windowsInstaller, bool) {
	var best windowsInstaller
	found := false
	for _, key := range keys {
		item, ok := parseWindowsInstallerKey(key)
		if !ok {
			continue
		}
		if !found || compareSemverish(item.Version, best.Version) > 0 {
			best = item
			found = true
		}
	}
	return best, found
}

func compareSemverish(a, b string) int {
	as := strings.Split(a, ".")
	bs := strings.Split(b, ".")
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var ai, bi int
		if i < len(as) {
			ai, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			bi, _ = strconv.Atoi(bs[i])
		}
		if ai == bi {
			continue
		}
		if ai > bi {
			return 1
		}
		return -1
	}
	return 0
}
