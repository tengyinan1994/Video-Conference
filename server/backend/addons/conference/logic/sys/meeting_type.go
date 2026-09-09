package sys

import (
	"context"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
)

type sSysMeetingType struct{}

func NewSysMeetingType() *sSysMeetingType {
	return &sSysMeetingType{}
}

func init() {
	service.RegisterSysMeetingType(NewSysMeetingType())
}

func meetingTypeModel(ctx context.Context) *gdb.Model {
	return g.DB().Model(consts.MeetingTypeTable).Safe().Ctx(ctx)
}

func (s *sSysMeetingType) Options(ctx context.Context) (list []*sysin.MeetingTypeOptionModel, err error) {
	var rows []*entity.MeetingType
	if err = meetingTypeModel(ctx).OrderAsc("id").Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询会议类型失败")
	}
	list = make([]*sysin.MeetingTypeOptionModel, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		list = append(list, &sysin.MeetingTypeOptionModel{Id: row.Id, Name: row.Name})
	}
	return
}

func (s *sSysMeetingType) NamesByIDs(ctx context.Context, ids []int64) (names map[int64]string, err error) {
	names = make(map[int64]string)
	if len(ids) == 0 {
		return
	}
	var rows []*entity.MeetingType
	if err = meetingTypeModel(ctx).WhereIn("id", ids).Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询会议类型失败")
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		names[row.Id] = row.Name
	}
	return
}

func (s *sSysMeetingType) AssertExists(ctx context.Context, id int64) (err error) {
	if id <= 0 {
		return nil
	}
	count, err := meetingTypeModel(ctx).Where("id", id).Count()
	if err != nil {
		return gerror.Wrap(err, "查询会议类型失败")
	}
	if count == 0 {
		return gerror.New("会议类型不存在")
	}
	return
}

// ========== 管理端 ==========

func (s *sSysMeetingType) AdminList(ctx context.Context, in *sysin.AdminMeetingTypeListInp) (list []*sysin.AdminMeetingTypeListModel, totalCount int, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	mod := meetingTypeModel(ctx)
	if in.Id > 0 {
		mod = mod.Where("id", in.Id)
	}
	if in.Name != "" {
		mod = mod.WhereLike("name", "%"+in.Name+"%")
	}
	mod = mod.Page(in.Page, in.PerPage).OrderAsc("id")

	var rows []*entity.MeetingType
	if err = mod.ScanAndCount(&rows, &totalCount, false); err != nil {
		return nil, 0, gerror.Wrap(err, "查询会议类型列表失败")
	}

	list = make([]*sysin.AdminMeetingTypeListModel, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		list = append(list, toAdminMeetingTypeItem(row))
	}
	return
}

func (s *sSysMeetingType) AdminView(ctx context.Context, in *sysin.AdminMeetingTypeViewInp) (res *sysin.AdminMeetingTypeViewModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	var row *entity.MeetingType
	if err = meetingTypeModel(ctx).Where("id", in.Id).Scan(&row); err != nil {
		return nil, gerror.Wrap(err, "查询会议类型失败")
	}
	if row == nil {
		return nil, gerror.New("会议类型不存在")
	}
	res = &sysin.AdminMeetingTypeViewModel{AdminMeetingTypeListModel: toAdminMeetingTypeItem(row)}
	return
}

func (s *sSysMeetingType) AdminEdit(ctx context.Context, in *sysin.AdminMeetingTypeEditInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}

	now := gtime.Now()

	// 编辑
	if in.Id > 0 {
		var row *entity.MeetingType
		if err = meetingTypeModel(ctx).Where("id", in.Id).Scan(&row); err != nil {
			return gerror.Wrap(err, "查询会议类型失败")
		}
		if row == nil {
			return gerror.New("会议类型不存在")
		}
		if err = s.assertNameUnique(ctx, in.Name, in.Id); err != nil {
			return
		}
		if _, err = meetingTypeModel(ctx).Where("id", in.Id).Data(g.Map{
			"name":       in.Name,
			"updated_at": now,
		}).Update(); err != nil {
			return gerror.Wrap(err, "更新会议类型失败")
		}
		return
	}

	// 新增
	if err = s.assertNameUnique(ctx, in.Name, 0); err != nil {
		return
	}
	if _, err = meetingTypeModel(ctx).Data(g.Map{
		"name":       in.Name,
		"created_at": now,
		"updated_at": now,
	}).Insert(); err != nil {
		return gerror.Wrap(err, "创建会议类型失败")
	}
	return
}

func (s *sSysMeetingType) AdminDelete(ctx context.Context, in *sysin.AdminMeetingTypeDeleteInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	ids := gconv.Int64s(in.Id)
	if len(ids) == 0 {
		return gerror.New("类型ID不能为空")
	}

	// 已被会议引用时拒绝删除，避免出现悬空类型
	count, err := g.DB().Model(consts.MeetingTable).Safe().Ctx(ctx).
		WhereIn("type_id", ids).
		Count()
	if err != nil {
		return gerror.Wrap(err, "查询会议使用情况失败")
	}
	if count > 0 {
		return gerror.New("该会议类型已被会议使用，无法删除")
	}

	if _, err = meetingTypeModel(ctx).WhereIn("id", ids).Delete(); err != nil {
		return gerror.Wrap(err, "删除会议类型失败")
	}
	return
}

// assertNameUnique 类型名称唯一校验；excludeId 用于编辑时排除自身
func (s *sSysMeetingType) assertNameUnique(ctx context.Context, name string, excludeId int64) (err error) {
	mod := meetingTypeModel(ctx).Where("name", name)
	if excludeId > 0 {
		mod = mod.WhereNot("id", excludeId)
	}
	count, err := mod.Count()
	if err != nil {
		return gerror.Wrap(err, "查询会议类型失败")
	}
	if count > 0 {
		return gerror.New("会议类型名称已存在")
	}
	return
}

func toAdminMeetingTypeItem(row *entity.MeetingType) *sysin.AdminMeetingTypeListModel {
	return &sysin.AdminMeetingTypeListModel{
		Id:        row.Id,
		Name:      row.Name,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
