package api

import (
	"context"

	"hotgo/addons/conference/api/api/auth"
	"hotgo/addons/conference/service"
)

// AuthPublic 公开认证接口（无需登录）
var AuthPublic = cAuthPublic{}

type cAuthPublic struct{}

func (c *cAuthPublic) Captcha(ctx context.Context, _ *auth.CaptchaReq) (res *auth.CaptchaRes, err error) {
	data, err := service.SysAuth().Captcha(ctx)
	if err != nil {
		return
	}
	res = &auth.CaptchaRes{AuthCaptchaModel: data}
	return
}

func (c *cAuthPublic) LoginConfig(ctx context.Context, _ *auth.LoginConfigReq) (res *auth.LoginConfigRes, err error) {
	data, err := service.SysAuth().LoginConfig(ctx)
	if err != nil {
		return
	}
	res = &auth.LoginConfigRes{AuthLoginConfigModel: data}
	return
}

func (c *cAuthPublic) Login(ctx context.Context, req *auth.LoginReq) (res *auth.LoginRes, err error) {
	data, err := service.SysAuth().Login(ctx, &req.AuthLoginInp)
	if err != nil {
		return
	}
	res = &auth.LoginRes{AuthLoginModel: data}
	return
}

// Auth 需登录的认证接口
var Auth = cAuth{}

type cAuth struct{}

func (c *cAuth) Logout(ctx context.Context, _ *auth.LogoutReq) (res *auth.LogoutRes, err error) {
	err = service.SysAuth().Logout(ctx)
	res = new(auth.LogoutRes)
	return
}

func (c *cAuth) Me(ctx context.Context, _ *auth.MeReq) (res *auth.MeRes, err error) {
	data, err := service.SysAuth().Me(ctx)
	if err != nil {
		return
	}
	res = &auth.MeRes{AuthMeModel: data}
	return
}
