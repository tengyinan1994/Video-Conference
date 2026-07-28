package token

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

// CreateReq 创建会议 Token
type CreateReq struct {
	g.Meta `path:"/token/create" method:"post" tags:"视频会议" summary:"创建进房 Token"`
	sysin.TokenCreateInp
}

type CreateRes struct {
	*sysin.TokenCreateModel
}
