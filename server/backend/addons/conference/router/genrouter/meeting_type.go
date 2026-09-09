package genrouter

import "hotgo/addons/conference/controller/admin/sys"

func init() {
	LoginRequiredRouter = append(LoginRequiredRouter, sys.MeetingType) // 会议类型
}
