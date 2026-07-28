package api

import (
	"context"

	"hotgo/addons/conference/api/api/token"
	"hotgo/addons/conference/service"
)

var (
	Token = cToken{}
)

type cToken struct{}

func (c *cToken) Create(ctx context.Context, req *token.CreateReq) (res *token.CreateRes, err error) {
	data, err := service.SysToken().Create(ctx, &req.TokenCreateInp)
	if err != nil {
		return
	}
	res = new(token.CreateRes)
	res.TokenCreateModel = data
	return
}
