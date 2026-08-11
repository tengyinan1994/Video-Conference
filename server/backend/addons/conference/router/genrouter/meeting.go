package genrouter

import "hotgo/addons/conference/controller/admin/sys"

func init() {
	LoginRequiredRouter = append(LoginRequiredRouter, sys.Meeting) // 会议管理
}
