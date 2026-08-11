// Package captcha
// @Link  https://github.com/bufanyun/hotgo
// @Copyright  Copyright (c) 2023 HotGo CLI
// @License  https://github.com/bufanyun/hotgo/blob/master/LICENSE
package captcha

import (
	"context"
	"hotgo/internal/consts"
	"image/color"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/mojocn/base64Captcha"
)

// store 验证码存储方式
var store = base64Captcha.DefaultMemStore

// Generate 生成验证码
func Generate(ctx context.Context, captchaType int) (id string, base64 string) {
	var err error

	// 与深色玻璃登录面板更协调的低饱和底色
	bg := &color.RGBA{R: 42, G: 52, B: 68, A: 255}

	switch captchaType {
	// 算数
	case consts.CaptchaTypeMath:
		driver := &base64Captcha.DriverMath{
			Height:          40,
			Width:           100,
			NoiseCount:      0,
			ShowLineOptions: 0,
			BgColor:         bg,
			Fonts:           []string{"chromohv.ttf"},
		}
		c := base64Captcha.NewCaptcha(driver.ConvertFonts(), store)
		id, base64, _, err = c.Generate()
	// 字符
	default:
		driver := &base64Captcha.DriverString{
			Height:          40,
			Width:           100,
			NoiseCount:      0,
			ShowLineOptions: 0,
			Length:          4,
			BgColor:         bg,
			Source:          "ABCDEFGHJKLMNPQRSTUVWXYZ23456789",
			Fonts:           []string{"chromohv.ttf"},
		}
		c := base64Captcha.NewCaptcha(driver.ConvertFonts(), store)
		id, base64, _, err = c.Generate()
	}

	if err != nil {
		g.Log().Errorf(ctx, "captcha.Generate err:%+v", err)
	}

	return
}

// Verify 验证输入的验证码是否正确
func Verify(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	return store.Verify(id, gstr.ToLower(answer), true)
}
