package minutes

import (
	"hotgo/addons/conference/model/input/sysin"

	"github.com/gogf/gf/v2/frame/g"
)

type ViewReq struct {
	g.Meta `path:"/minutes/view" method:"get" tags:"会议纪要" summary:"查询会后AI纪要"`
	sysin.MinutesViewInp
}

type ViewRes struct {
	*sysin.MinutesModel
}

type RegenerateReq struct {
	g.Meta `path:"/minutes/regenerate" method:"post" tags:"会议纪要" summary:"重新生成会后AI纪要"`
	sysin.MinutesRegenerateInp
}

type RegenerateRes struct {
	*sysin.MinutesModel
}
