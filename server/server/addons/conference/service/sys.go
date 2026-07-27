package service

import (
	"context"

	"hotgo/addons/conference/model/input/sysin"
)

type (
	ISysToken interface {
		Create(ctx context.Context, in *sysin.TokenCreateInp) (res *sysin.TokenCreateModel, err error)
	}
)

var (
	localSysToken ISysToken
)

func SysToken() ISysToken {
	if localSysToken == nil {
		panic("implement not found for interface ISysToken, forgot register?")
	}
	return localSysToken
}

func RegisterSysToken(i ISysToken) {
	localSysToken = i
}
