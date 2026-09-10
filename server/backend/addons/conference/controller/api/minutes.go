package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	"hotgo/addons/conference/api/api/minutes"
	"hotgo/addons/conference/model"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// Minutes 需登录的纪要接口
var Minutes = cMinutes{}

type cMinutes struct{}

func (c *cMinutes) View(ctx context.Context, req *minutes.ViewReq) (res *minutes.ViewRes, err error) {
	data, err := service.SysMinutes().View(ctx, &req.MinutesViewInp)
	if err != nil {
		return
	}
	res = &minutes.ViewRes{MinutesModel: data}
	return
}

func (c *cMinutes) Regenerate(ctx context.Context, req *minutes.RegenerateReq) (res *minutes.RegenerateRes, err error) {
	data, err := service.SysMinutes().Regenerate(ctx, &req.MinutesRegenerateInp)
	if err != nil {
		return
	}
	res = &minutes.RegenerateRes{MinutesModel: data}
	return
}

// HandleMinutesCallback Worker 回调（验签，无用户登录）
func HandleMinutesCallback(r *ghttp.Request) {
	ctx := r.Context()
	body := r.GetBody()
	if len(body) > 0 {
		r.Request.Body = io.NopCloser(bytes.NewReader(body))
	}

	cfg, err := readMinutesConfig(ctx)
	if err != nil {
		g.Log().Warningf(ctx, "minutes callback: load config failed: %+v", err)
		r.Response.WriteStatus(500)
		return
	}
	sig := r.Header.Get("X-Minutes-Signature")
	if !verifyMinutesHMAC(cfg.CallbackSecret, sig, body) {
		g.Log().Warningf(ctx, "minutes callback: bad signature")
		r.Response.WriteStatus(401)
		return
	}

	var in sysin.MinutesCallbackInp
	if err = gjson.New(body).Scan(&in); err != nil {
		g.Log().Warningf(ctx, "minutes callback: bad json: %+v", err)
		r.Response.WriteStatus(400)
		return
	}
	if err = service.SysMinutes().Callback(ctx, &in); err != nil {
		g.Log().Warningf(ctx, "minutes callback: handle failed: %+v", err)
		r.Response.WriteStatus(500)
		return
	}
	r.Response.WriteStatus(200)
	r.Response.WriteJson(g.Map{"ok": true})
}

func readMinutesConfig(ctx context.Context) (*model.MinutesConfig, error) {
	cfg := &model.MinutesConfig{Enabled: true, MinTranscriptChars: 8}
	v, err := g.Cfg().Get(ctx, "minutes")
	if err != nil {
		return nil, gerror.Wrap(err, "读取 minutes 配置失败")
	}
	if v != nil && !v.IsNil() && !v.IsEmpty() {
		if err = v.Scan(cfg); err != nil {
			return nil, gerror.Wrap(err, "解析 minutes 配置失败")
		}
	}
	return cfg, nil
}

func verifyMinutesHMAC(secret, sig string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	sig = strings.TrimSpace(sig)
	if secret == "" {
		return true
	}
	if sig == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expect := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expect), []byte(sig))
}
