package auth

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

type CaptchaReq struct {
	g.Meta `path:"/auth/captcha" method:"get" tags:"视频会议认证" summary:"获取登录验证码"`
}

type CaptchaRes struct {
	*sysin.AuthCaptchaModel
}

type LoginConfigReq struct {
	g.Meta `path:"/auth/loginConfig" method:"get" tags:"视频会议认证" summary:"获取登录配置"`
}

type LoginConfigRes struct {
	*sysin.AuthLoginConfigModel
}

type LoginReq struct {
	g.Meta `path:"/auth/login" method:"post" tags:"视频会议认证" summary:"账号登录"`
	sysin.AuthLoginInp
}

type LoginRes struct {
	*sysin.AuthLoginModel
}

type LogoutReq struct {
	g.Meta `path:"/auth/logout" method:"post" tags:"视频会议认证" summary:"注销登录"`
}

type LogoutRes struct{}

type MeReq struct {
	g.Meta `path:"/auth/me" method:"get" tags:"视频会议认证" summary:"当前用户"`
}

type MeRes struct {
	*sysin.AuthMeModel
}
