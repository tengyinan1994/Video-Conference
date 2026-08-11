package sysin

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
)

// AuthLoginInp 会议客户端登录
type AuthLoginInp struct {
	Username string `json:"username" v:"required#用户名不能为空" dc:"用户名"`
	Password string `json:"password" v:"required#密码不能为空" dc:"密码（AES加密后）"`
	Cid      string `json:"cid" dc:"验证码ID"`
	Code     string `json:"code" dc:"验证码"`
}

func (in *AuthLoginInp) Filter(ctx context.Context) (err error) {
	in.Username = strings.TrimSpace(in.Username)
	in.Cid = strings.TrimSpace(in.Cid)
	in.Code = strings.TrimSpace(in.Code)
	if in.Username == "" {
		return gerror.New("用户名不能为空")
	}
	if in.Password == "" {
		return gerror.New("密码不能为空")
	}
	return
}

// AuthLoginModel 登录结果
type AuthLoginModel struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	Token    string `json:"token"`
	Expires  int64  `json:"expires"`
}

// AuthMeModel 当前用户
type AuthMeModel struct {
	Id       int64  `json:"id"`
	Username string `json:"username"`
	RealName string `json:"realName"`
}

// AuthCaptchaModel 验证码
type AuthCaptchaModel struct {
	Cid    string `json:"cid"`
	Base64 string `json:"base64"`
}

// AuthLoginConfigModel 登录配置（精简）
type AuthLoginConfigModel struct {
	CaptchaSwitch int    `json:"captchaSwitch"`
	ProjectName   string `json:"projectName"`
}
