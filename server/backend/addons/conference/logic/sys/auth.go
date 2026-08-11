package sys

import (
	"context"

	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/consts"
	"hotgo/internal/library/captcha"
	"hotgo/internal/library/contexts"
	"hotgo/internal/library/token"
	"hotgo/internal/model/input/adminin"
	iservice "hotgo/internal/service"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/i18n/gi18n"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
)

type sSysAuth struct{}

func NewSysAuth() *sSysAuth {
	return &sSysAuth{}
}

func init() {
	service.RegisterSysAuth(NewSysAuth())
}

func (s *sSysAuth) Captcha(ctx context.Context) (res *sysin.AuthCaptchaModel, err error) {
	loginConf, err := iservice.SysConfig().GetLogin(ctx)
	if err != nil {
		return
	}
	cid, base64 := captcha.Generate(ctx, loginConf.CaptchaType)
	res = &sysin.AuthCaptchaModel{Cid: cid, Base64: base64}
	return
}

func (s *sSysAuth) LoginConfig(ctx context.Context) (res *sysin.AuthLoginConfigModel, err error) {
	login, err := iservice.SysConfig().GetLogin(ctx)
	if err != nil {
		return
	}
	res = &sysin.AuthLoginConfigModel{
		CaptchaSwitch: login.CaptchaSwitch,
		ProjectName:   gi18n.T(ctx, "视频会议系统"),
	}
	return
}

func (s *sSysAuth) Login(ctx context.Context, in *sysin.AuthLoginInp) (res *sysin.AuthLoginModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	loginConf, err := iservice.SysConfig().GetLogin(ctx)
	if err != nil {
		return
	}
	if loginConf.CaptchaSwitch == consts.StatusEnabled {
		if !captcha.Verify(in.Cid, in.Code) {
			return nil, gerror.New("验证码错误")
		}
	}

	model, err := iservice.AdminSite().AccountLogin(ctx, &adminin.AccountLoginInp{
		Username: in.Username,
		Password: in.Password,
		Cid:      in.Cid,
		Code:     in.Code,
	})
	if err != nil {
		return
	}
	res = new(sysin.AuthLoginModel)
	if err = gconv.Scan(model, res); err != nil {
		return nil, gerror.Wrap(err, "登录结果解析失败")
	}
	return
}

func (s *sSysAuth) Logout(ctx context.Context) (err error) {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return gerror.New("无效请求")
	}
	return token.Logout(r)
}

func (s *sSysAuth) Me(ctx context.Context) (res *sysin.AuthMeModel, err error) {
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	res = &sysin.AuthMeModel{
		Id:       user.Id,
		Username: user.Username,
		RealName: user.RealName,
	}
	if res.RealName == "" {
		res.RealName = user.Username
	}
	return
}
