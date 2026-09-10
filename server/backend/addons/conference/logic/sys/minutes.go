package sys

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model"
	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/contexts"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

type sSysMinutes struct{}

func NewSysMinutes() *sSysMinutes {
	return &sSysMinutes{}
}

func init() {
	service.RegisterSysMinutes(NewSysMinutes())
}

func minutesModel(ctx context.Context) *gdb.Model {
	return g.DB().Model(consts.MinutesTable).Safe().Ctx(ctx)
}

func loadMinutesConfig(ctx context.Context) (*model.MinutesConfig, error) {
	cfg := &model.MinutesConfig{
		Enabled:            true,
		MinTranscriptChars: 8,
	}
	v, err := g.Cfg().Get(ctx, "minutes")
	if err != nil {
		return cfg, nil
	}
	if v == nil || v.IsNil() || v.IsEmpty() {
		return cfg, nil
	}
	if err = v.Scan(cfg); err != nil {
		return nil, gerror.Wrap(err, "读取 minutes 配置失败")
	}
	if cfg.MinTranscriptChars <= 0 {
		cfg.MinTranscriptChars = 8
	}
	return cfg, nil
}

func (s *sSysMinutes) View(ctx context.Context, in *sysin.MinutesViewInp) (res *sysin.MinutesModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	row, err := s.getByMeetingID(ctx, in.MeetingId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &sysin.MinutesModel{
			MeetingId: in.MeetingId,
			Status:    consts.MinutesStatusUnavailable,
		}, nil
	}
	return toMinutesModel(row), nil
}

func (s *sSysMinutes) Regenerate(ctx context.Context, in *sysin.MinutesRegenerateInp) (res *sysin.MinutesModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	meeting := &entity.Meeting{}
	if err = g.DB().Model(consts.MeetingTable).Ctx(ctx).Where("id", in.MeetingId).Scan(meeting); err != nil {
		return nil, gerror.Wrap(err, "查询会议失败")
	}
	if meeting.Id == 0 {
		return nil, gerror.New("会议不存在")
	}
	if meeting.HostId != user.Id {
		return nil, gerror.New("仅主持人可重新生成纪要")
	}
	if !isEndedStatus(meeting.Status) {
		return nil, gerror.New("仅已结束会议可生成纪要")
	}
	if err = s.TryEnqueue(ctx, in.MeetingId, true); err != nil {
		return nil, err
	}
	row, err := s.getByMeetingID(ctx, in.MeetingId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return &sysin.MinutesModel{MeetingId: in.MeetingId, Status: consts.MinutesStatusPending}, nil
	}
	return toMinutesModel(row), nil
}

func (s *sSysMinutes) Callback(ctx context.Context, in *sysin.MinutesCallbackInp) (err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	now := gtime.Now()
	data := g.Map{
		"status":     in.Status,
		"updated_at": now,
		"error_msg":  in.ErrorMsg,
		"model":      in.Model,
	}
	switch in.Status {
	case consts.MinutesStatusTranscribing, consts.MinutesStatusSummarizing, consts.MinutesStatusPending:
		// 进度态
	case consts.MinutesStatusReady:
		data["transcript"] = in.Transcript
		data["summary"] = in.Summary
		data["structured"] = in.Structured
		data["source_recording_ids"] = gjson.New(in.SourceRecordingIds)
		data["generated_at"] = now
		data["error_msg"] = ""
	case consts.MinutesStatusSkippedEmpty, consts.MinutesStatusFailed, consts.MinutesStatusUnavailable:
		if in.Transcript != "" {
			data["transcript"] = in.Transcript
		}
		if in.Summary != "" {
			data["summary"] = in.Summary
		}
		if in.Structured != nil {
			data["structured"] = in.Structured
		}
		if len(in.SourceRecordingIds) > 0 {
			data["source_recording_ids"] = gjson.New(in.SourceRecordingIds)
		}
		data["generated_at"] = now
	default:
		return gerror.Newf("不支持的纪要状态: %s", in.Status)
	}

	row, err := s.getByMeetingID(ctx, in.MeetingId)
	if err != nil {
		return err
	}
	if row == nil {
		data["meeting_id"] = in.MeetingId
		data["created_at"] = now
		_, err = minutesModel(ctx).Data(data).Insert()
		return err
	}
	_, err = minutesModel(ctx).Where("meeting_id", in.MeetingId).Data(data).Update()
	return err
}

// TryEnqueue 会议结束后且 AI 音源就绪时派单；force 时强制重跑。
func (s *sSysMinutes) TryEnqueue(ctx context.Context, meetingId int64, force bool) error {
	if meetingId <= 0 {
		return nil
	}
	cfg, err := loadMinutesConfig(ctx)
	if err != nil {
		return err
	}
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	meeting := &entity.Meeting{}
	if err = g.DB().Model(consts.MeetingTable).Ctx(ctx).Where("id", meetingId).Scan(meeting); err != nil {
		return gerror.Wrap(err, "查询会议失败")
	}
	if meeting.Id == 0 {
		return gerror.New("会议不存在")
	}
	if !isEndedStatus(meeting.Status) {
		return nil
	}

	existing, err := s.getByMeetingID(ctx, meetingId)
	if err != nil {
		return err
	}
	if existing != nil && !force && minutesSkipEnqueue(existing) {
		return nil
	}

	var aiRows []*entity.Recording
	if err = recordingModel(ctx).
		Where("meeting_id", meetingId).
		Where("purpose", consts.RecordingPurposeAI).
		OrderAsc("seq").
		Scan(&aiRows); err != nil {
		return gerror.Wrap(err, "查询 AI 音源失败")
	}

	var complete []*entity.Recording
	inFlight := false
	for _, r := range aiRows {
		if r == nil {
			continue
		}
		switch r.Status {
		case consts.RecordingStatusStarting, consts.RecordingStatusActive, consts.RecordingStatusStopping:
			inFlight = true
		case consts.RecordingStatusComplete:
			key := strings.TrimSpace(r.ObjectKey)
			if key != "" && !strings.Contains(key, "{") {
				complete = append(complete, r)
			}
		}
	}
	now := gtime.Now()
	if inFlight {
		// 会议刚结束、音源还在收尾：先落 pending；webhook 终态会再入队
		return s.upsertStatus(ctx, meetingId, consts.MinutesStatusPending, "", now)
	}

	if len(complete) == 0 {
		return s.upsertStatus(ctx, meetingId, consts.MinutesStatusUnavailable, "无可用的 AI 音源文件", now)
	}

	if err = s.upsertStatus(ctx, meetingId, consts.MinutesStatusPending, "", now); err != nil {
		return err
	}

	if strings.TrimSpace(cfg.WorkerUrl) == "" {
		return s.upsertStatus(ctx, meetingId, consts.MinutesStatusFailed, "未配置 minutes.workerUrl", now)
	}

	attendees := attendeesFromJSON(meeting.Attendees)
	segments := make([]g.Map, 0, len(complete))
	ids := make([]int64, 0, len(complete))
	for _, r := range complete {
		ids = append(ids, r.Id)
		segments = append(segments, g.Map{
			"id":        r.Id,
			"seq":       r.Seq,
			"objectKey": r.ObjectKey,
			"fileSize":  r.FileSize,
		})
	}

	payload := g.Map{
		"meetingId":          meetingId,
		"title":              meeting.Title,
		"attendees":          attendees,
		"segments":           segments,
		"sourceRecordingIds": ids,
		"minTranscriptChars": cfg.MinTranscriptChars,
	}

	body, _ := json.Marshal(payload)
	url := strings.TrimRight(cfg.WorkerUrl, "/") + "/v1/jobs"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		_ = s.upsertStatus(ctx, meetingId, consts.MinutesStatusFailed, "创建派单请求失败", now)
		return gerror.Wrap(err, "创建纪要派单失败")
	}
	req.Header.Set("Content-Type", "application/json")
	if secret := strings.TrimSpace(cfg.CallbackSecret); secret != "" {
		req.Header.Set("X-Minutes-Signature", signMinutesBody(secret, body))
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_ = s.upsertStatus(ctx, meetingId, consts.MinutesStatusFailed, "Worker 不可达: "+err.Error(), now)
		g.Log().Warningf(ctx, "minutes dispatch failed meeting=%d err=%+v", meetingId, err)
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		msg := fmt.Sprintf("Worker 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		_ = s.upsertStatus(ctx, meetingId, consts.MinutesStatusFailed, msg, now)
		g.Log().Warningf(ctx, "minutes dispatch bad status meeting=%d %s", meetingId, msg)
		return nil
	}
	_, _ = minutesModel(ctx).Where("meeting_id", meetingId).Data(g.Map{
		"source_recording_ids": gjson.New(ids),
		"updated_at":           now,
	}).Update()
	return nil
}

func (s *sSysMinutes) ListByMeetingIDs(ctx context.Context, meetingIDs []int64) (map[int64]*sysin.MinutesModel, error) {
	out := make(map[int64]*sysin.MinutesModel)
	if len(meetingIDs) == 0 {
		return out, nil
	}
	var rows []*entity.Minutes
	if err := minutesModel(ctx).WhereIn("meeting_id", meetingIDs).Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询纪要失败")
	}
	for _, r := range rows {
		if r == nil {
			continue
		}
		out[r.MeetingId] = toMinutesModel(r)
	}
	return out, nil
}

func (s *sSysMinutes) getByMeetingID(ctx context.Context, meetingId int64) (*entity.Minutes, error) {
	var row *entity.Minutes
	if err := minutesModel(ctx).Where("meeting_id", meetingId).Scan(&row); err != nil {
		return nil, gerror.Wrap(err, "查询纪要失败")
	}
	if row == nil || row.Id == 0 {
		return nil, nil
	}
	return row, nil
}

func (s *sSysMinutes) upsertStatus(ctx context.Context, meetingId int64, status, errMsg string, now *gtime.Time) error {
	row, err := s.getByMeetingID(ctx, meetingId)
	if err != nil {
		return err
	}
	data := g.Map{
		"status":     status,
		"error_msg":  errMsg,
		"updated_at": now,
	}
	if row == nil {
		data["meeting_id"] = meetingId
		data["created_at"] = now
		_, err = minutesModel(ctx).Data(data).Insert()
		return err
	}
	_, err = minutesModel(ctx).Where("meeting_id", meetingId).Data(data).Update()
	return err
}

func minutesSkipEnqueue(row *entity.Minutes) bool {
	if row == nil {
		return false
	}
	switch row.Status {
	case consts.MinutesStatusTranscribing, consts.MinutesStatusSummarizing, consts.MinutesStatusReady:
		return true
	case consts.MinutesStatusPending:
		// 仅「已经派过单」的 pending 才跳过；会结束时先落 pending、音源尚未就绪的必须再入队
		if row.SourceRecordingIds == nil || row.SourceRecordingIds.IsNil() || len(row.SourceRecordingIds.Array()) == 0 {
			return false
		}
		return true
	default:
		return false
	}
}

func toMinutesModel(r *entity.Minutes) *sysin.MinutesModel {
	if r == nil {
		return nil
	}
	ids := make([]int64, 0)
	if r.SourceRecordingIds != nil && !r.SourceRecordingIds.IsNil() {
		_ = r.SourceRecordingIds.Scan(&ids)
	}
	return &sysin.MinutesModel{
		MeetingId:          r.MeetingId,
		Status:             r.Status,
		Transcript:         r.Transcript,
		Summary:            r.Summary,
		Structured:         r.Structured,
		SourceRecordingIds: ids,
		ErrorMsg:           r.ErrorMsg,
		Model:              r.Model,
		GeneratedAt:        r.GeneratedAt,
	}
}

func signMinutesBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyMinutesSignature 校验 Worker 回调签名
func VerifyMinutesSignature(secret, sig string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	sig = strings.TrimSpace(sig)
	if secret == "" {
		return true
	}
	if sig == "" {
		return false
	}
	expect := signMinutesBody(secret, body)
	return hmac.Equal([]byte(expect), []byte(sig))
}
