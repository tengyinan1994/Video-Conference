package sys

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"hotgo/addons/conference/consts"
	"hotgo/addons/conference/model"
	"hotgo/addons/conference/model/entity"
	"hotgo/addons/conference/model/input/sysin"
	"hotgo/addons/conference/service"
	"hotgo/internal/library/contexts"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/livekit/protocol/livekit"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type sSysRecording struct{}

func NewSysRecording() *sSysRecording {
	return &sSysRecording{}
}

func init() {
	service.RegisterSysRecording(NewSysRecording())
}

func recordingModel(ctx context.Context) *gdb.Model {
	return g.DB().Model(consts.RecordingTable).Safe().Ctx(ctx)
}

var (
	bucketOnce sync.Once
	bucketErr  error
)

func (s *sSysRecording) Start(ctx context.Context, in *sysin.RecordingStartInp) (res *sysin.RecordingStartModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	meeting, err := service.SysMeeting().GetByRoomName(ctx, in.Room)
	if err != nil {
		return
	}
	if meeting == nil {
		return nil, gerror.New("会议室不存在")
	}
	if meeting.HostId != user.Id {
		return nil, gerror.New("仅主持人可开始录制")
	}
	if err = service.SysMeeting().AssertJoinable(ctx, meeting); err != nil {
		return
	}
	seg, err := s.startSegment(ctx, meeting, user.Id)
	if err != nil {
		return
	}
	res = &sysin.RecordingStartModel{RecordingSegmentModel: toRecordingSegment(ctx, seg)}
	return
}

func (s *sSysRecording) Stop(ctx context.Context, in *sysin.RecordingStopInp) (res *sysin.RecordingStopModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, gerror.New("请先登录")
	}
	meeting, err := service.SysMeeting().GetByRoomName(ctx, in.Room)
	if err != nil {
		return
	}
	if meeting == nil {
		return nil, gerror.New("会议室不存在")
	}
	if meeting.HostId != user.Id {
		return nil, gerror.New("仅主持人可停止录制")
	}
	seg, err := s.stopActiveSegment(ctx, meeting.Id, meeting.RoomName)
	if err != nil {
		return
	}
	res = &sysin.RecordingStopModel{RecordingSegmentModel: toRecordingSegment(ctx, seg)}
	return
}

func (s *sSysRecording) Status(ctx context.Context, in *sysin.RecordingStatusInp) (res *sysin.RecordingStatusModel, err error) {
	if err = in.Filter(ctx); err != nil {
		return
	}
	var meeting *entity.Meeting
	if in.MeetingId > 0 {
		meeting = &entity.Meeting{}
		if err = g.DB().Model(consts.MeetingTable).Ctx(ctx).Where("id", in.MeetingId).Scan(meeting); err != nil {
			return nil, gerror.Wrap(err, "查询会议失败")
		}
		if meeting.Id == 0 {
			return nil, gerror.New("会议不存在")
		}
	} else {
		meeting, err = service.SysMeeting().GetByRoomName(ctx, in.Room)
		if err != nil {
			return
		}
		if meeting == nil {
			return nil, gerror.New("会议室不存在")
		}
	}

	var rows []*entity.Recording
	if err = recordingModel(ctx).Where("meeting_id", meeting.Id).OrderAsc("seq").Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询录制记录失败")
	}
	segments := make([]*sysin.RecordingSegmentModel, 0, len(rows))
	active := false
	for _, r := range rows {
		if r == nil {
			continue
		}
		segments = append(segments, toRecordingSegment(ctx, r))
		if r.Status == consts.RecordingStatusStarting ||
			r.Status == consts.RecordingStatusActive {
			active = true
		}
	}
	res = &sysin.RecordingStatusModel{Active: active, Segments: segments}
	return
}

// TryAutoStart 主持人进房且会议开启了录制时，若无进行中段则自动起第一段
func (s *sSysRecording) TryAutoStart(ctx context.Context, meeting *entity.Meeting, startedBy int64) error {
	if meeting == nil || meeting.RecordEnabled == 0 || meeting.Id <= 0 {
		return nil
	}
	if hasActiveRecording(ctx, meeting.Id) {
		return nil
	}
	_, err := s.startSegment(ctx, meeting, startedBy)
	if err != nil {
		g.Log().Warningf(ctx, "conference auto-start recording failed meeting=%d room=%s err=%+v", meeting.Id, meeting.RoomName, err)
		return err
	}
	return nil
}

func (s *sSysRecording) ListByMeetingIDs(ctx context.Context, meetingIDs []int64) (map[int64][]*sysin.RecordingSegmentModel, error) {
	out := make(map[int64][]*sysin.RecordingSegmentModel)
	if len(meetingIDs) == 0 {
		return out, nil
	}
	var rows []*entity.Recording
	if err := recordingModel(ctx).WhereIn("meeting_id", meetingIDs).OrderAsc("seq").Scan(&rows); err != nil {
		return nil, gerror.Wrap(err, "查询录制记录失败")
	}
	for _, r := range rows {
		if r == nil {
			continue
		}
		out[r.MeetingId] = append(out[r.MeetingId], toRecordingSegment(ctx, r))
	}
	return out, nil
}

// StopAllForMeeting 结束会议时停止进行中的录制
func (s *sSysRecording) StopAllForMeeting(ctx context.Context, meetingId int64, roomName string) {
	if meetingId <= 0 {
		return
	}
	if _, err := s.stopActiveSegment(ctx, meetingId, roomName); err != nil {
		// 无进行中段不算错误
		if strings.Contains(err.Error(), "当前没有进行中的录制") {
			return
		}
		g.Log().Warningf(ctx, "conference stop recording on end failed meeting=%d err=%+v", meetingId, err)
	}
}

// HandleEgressWebhook 根据 Egress 事件回写分段状态
func (s *sSysRecording) HandleEgressWebhook(ctx context.Context, info *livekit.EgressInfo) {
	if info == nil || info.EgressId == "" {
		return
	}
	var row *entity.Recording
	if err := recordingModel(ctx).Where("egress_id", info.EgressId).Scan(&row); err != nil || row == nil {
		return
	}

	now := gtime.Now()
	data := g.Map{"updated_at": now}
	switch info.Status {
	case livekit.EgressStatus_EGRESS_STARTING:
		data["status"] = consts.RecordingStatusStarting
	case livekit.EgressStatus_EGRESS_ACTIVE:
		data["status"] = consts.RecordingStatusActive
	case livekit.EgressStatus_EGRESS_ENDING:
		data["status"] = consts.RecordingStatusStopping
	case livekit.EgressStatus_EGRESS_COMPLETE:
		data["status"] = consts.RecordingStatusComplete
		data["ended_at"] = now
		if len(info.FileResults) > 0 && info.FileResults[0] != nil {
			fr := info.FileResults[0]
			if fr.Filename != "" {
				data["object_key"] = fr.Filename
			}
			if fr.Size > 0 {
				data["file_size"] = fr.Size
			}
		}
	case livekit.EgressStatus_EGRESS_FAILED, livekit.EgressStatus_EGRESS_ABORTED, livekit.EgressStatus_EGRESS_LIMIT_REACHED:
		data["status"] = consts.RecordingStatusFailed
		data["ended_at"] = now
		if info.Error != "" {
			msg := info.Error
			if len(msg) > 500 {
				msg = msg[:500]
			}
			data["error_msg"] = msg
		}
	default:
		return
	}
	if _, err := recordingModel(ctx).Where("id", row.Id).Data(data).Update(); err != nil {
		g.Log().Warningf(ctx, "conference update recording from webhook failed egress=%s err=%+v", info.EgressId, err)
	}
}

func (s *sSysRecording) startSegment(ctx context.Context, meeting *entity.Meeting, startedBy int64) (*entity.Recording, error) {
	recCfg, err := loadRecordingConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !recCfg.Enabled {
		return nil, gerror.New("录制功能未启用")
	}
	if strings.TrimSpace(recCfg.S3.Endpoint) == "" || strings.TrimSpace(recCfg.S3.Bucket) == "" {
		return nil, gerror.New("录制存储未配置，请检查 recording.s3")
	}
	if hasActiveRecording(ctx, meeting.Id) {
		return nil, gerror.New("当前已有进行中的录制，请先停止")
	}

	if err = ensureRecordingBucket(ctx, recCfg); err != nil {
		g.Log().Warningf(ctx, "ensure recording bucket: %+v", err)
	}

	seq := nextRecordingSeq(ctx, meeting.Id)
	objectKey := fmt.Sprintf("%d/%d_{time}.mp4", meeting.Id, seq)

	client, _, _, err := newEgressClient(ctx)
	if err != nil {
		return nil, err
	}

	forcePath := recCfg.S3.ForcePathStyle
	req := &livekit.RoomCompositeEgressRequest{
		RoomName: meeting.RoomName,
		Layout:   "speaker",
		FileOutputs: []*livekit.EncodedFileOutput{{
			FileType: livekit.EncodedFileType_MP4,
			Filepath: objectKey,
			Output: &livekit.EncodedFileOutput_S3{
				S3: &livekit.S3Upload{
					AccessKey:      recCfg.S3.AccessKey,
					Secret:         recCfg.S3.SecretKey,
					Endpoint:       s3UploadEndpoint(recCfg),
					Bucket:         recCfg.S3.Bucket,
					Region:         recCfg.S3.Region,
					ForcePathStyle: forcePath,
				},
			},
		}},
	}

	info, err := client.StartRoomCompositeEgress(ctx, req)
	if err != nil {
		return nil, gerror.Wrap(err, "启动录制失败")
	}

	now := gtime.Now()
	status := consts.RecordingStatusStarting
	if info.Status == livekit.EgressStatus_EGRESS_ACTIVE {
		status = consts.RecordingStatusActive
	}
	id, err := recordingModel(ctx).Data(g.Map{
		"meeting_id": meeting.Id,
		"room_name":  meeting.RoomName,
		"egress_id":  info.EgressId,
		"seq":        seq,
		"status":     status,
		"object_key": objectKey,
		"started_at": now,
		"started_by": startedBy,
		"created_at": now,
		"updated_at": now,
	}).InsertAndGetId()
	if err != nil {
		// 尽力停掉已启动的 egress，避免孤儿任务
		_, _ = client.StopEgress(ctx, &livekit.StopEgressRequest{EgressId: info.EgressId})
		return nil, gerror.Wrap(err, "保存录制记录失败")
	}

	row := &entity.Recording{}
	if err = recordingModel(ctx).Where("id", id).Scan(row); err != nil {
		return nil, gerror.Wrap(err, "读取录制记录失败")
	}
	return row, nil
}

func (s *sSysRecording) stopActiveSegment(ctx context.Context, meetingId int64, roomName string) (*entity.Recording, error) {
	var row *entity.Recording
	err := recordingModel(ctx).
		Where("meeting_id", meetingId).
		WhereIn("status", g.Slice{consts.RecordingStatusStarting, consts.RecordingStatusActive}).
		OrderDesc("id").
		Limit(1).
		Scan(&row)
	if err != nil {
		return nil, gerror.Wrap(err, "查询进行中录制失败")
	}
	if row == nil || row.Id == 0 {
		return nil, gerror.New("当前没有进行中的录制")
	}

	if _, _, _, err = newEgressClient(ctx); err != nil {
		return nil, err
	}

	now := gtime.Now()
	if _, err = recordingModel(ctx).Where("id", row.Id).Data(g.Map{
		"status":     consts.RecordingStatusStopping,
		"updated_at": now,
	}).Update(); err != nil {
		return nil, gerror.Wrap(err, "更新录制状态失败")
	}
	stopEgressAsync(row.EgressId)
	_ = recordingModel(ctx).Where("id", row.Id).Scan(&row)
	_ = roomName
	return row, nil
}

func stopEgressAsync(egressId string) {
	if strings.TrimSpace(egressId) == "" {
		return
	}
	go func(id string) {
		bg := context.Background()
		client, _, _, err := newEgressClient(bg)
		if err != nil {
			g.Log().Warningf(bg, "conference async StopEgress client failed egress=%s err=%+v", id, err)
			return
		}
		if _, err = client.StopEgress(bg, &livekit.StopEgressRequest{EgressId: id}); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "not found") || strings.Contains(msg, "EGRESS_COMPLETE") {
				return
			}
			g.Log().Warningf(bg, "conference async StopEgress failed egress=%s err=%+v", id, err)
		}
	}(egressId)
}

func hasActiveRecording(ctx context.Context, meetingId int64) bool {
	count, err := recordingModel(ctx).
		Where("meeting_id", meetingId).
		WhereIn("status", g.Slice{
			consts.RecordingStatusStarting,
			consts.RecordingStatusActive,
		}).
		Count()
	if err != nil {
		return false
	}
	return count > 0
}

func nextRecordingSeq(ctx context.Context, meetingId int64) int {
	val, err := recordingModel(ctx).Where("meeting_id", meetingId).Max("seq")
	if err != nil {
		return 1
	}
	n := int(val)
	if n < 0 {
		return 1
	}
	return n + 1
}

func toRecordingSegment(ctx context.Context, r *entity.Recording) *sysin.RecordingSegmentModel {
	if r == nil {
		return nil
	}
	playURL, downloadURL := recordingAccessURLs(ctx, r)
	return &sysin.RecordingSegmentModel{
		Id:          r.Id,
		MeetingId:   r.MeetingId,
		RoomName:    r.RoomName,
		EgressId:    r.EgressId,
		Seq:         r.Seq,
		Status:      r.Status,
		ObjectKey:   r.ObjectKey,
		FileSize:    r.FileSize,
		StartedAt:   r.StartedAt,
		EndedAt:     r.EndedAt,
		ErrorMsg:    r.ErrorMsg,
		PlayUrl:     playURL,
		DownloadUrl: downloadURL,
	}
}

func recordingFileName(r *entity.Recording) string {
	if r == nil {
		return "recording.mp4"
	}
	return fmt.Sprintf("recording-%d-%d.mp4", r.MeetingId, r.Seq)
}

func recordingAccessURLs(ctx context.Context, r *entity.Recording) (playURL, downloadURL string) {
	if r == nil || r.Status != consts.RecordingStatusComplete {
		return "", ""
	}
	key := strings.TrimSpace(r.ObjectKey)
	if key == "" || strings.Contains(key, "{") {
		return "", ""
	}
	cfg, err := loadRecordingConfig(ctx)
	if err != nil {
		return "", ""
	}
	name := recordingFileName(r)
	playURL, err = presignRecordingObject(ctx, cfg, key, "inline", name)
	if err != nil {
		g.Log().Warningf(ctx, "presign recording play url failed key=%s err=%+v", key, err)
	}
	downloadURL, err = presignRecordingObject(ctx, cfg, key, "attachment", name)
	if err != nil {
		g.Log().Warningf(ctx, "presign recording download url failed key=%s err=%+v", key, err)
	}
	playURL = applyPublicEndpoint(playURL, cfg.PublicEndpoint)
	downloadURL = applyPublicEndpoint(downloadURL, cfg.PublicEndpoint)
	if downloadURL == "" {
		downloadURL = playURL
	}
	return playURL, downloadURL
}

func applyPublicEndpoint(raw, publicEndpoint string) string {
	raw = strings.TrimSpace(raw)
	publicEndpoint = strings.TrimSpace(publicEndpoint)
	if raw == "" || publicEndpoint == "" {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	pub, err := url.Parse(publicEndpoint)
	if err != nil || pub.Host == "" {
		return raw
	}
	u.Scheme = pub.Scheme
	u.Host = pub.Host
	return u.String()
}

func presignRecordingObject(ctx context.Context, cfg *model.RecordingConfig, objectKey, disposition, filename string) (string, error) {
	client, err := newRecordingS3Client(cfg)
	if err != nil {
		return "", err
	}
	params := url.Values{}
	if disposition != "" && filename != "" {
		params.Set("response-content-disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, filename))
	} else if disposition != "" {
		params.Set("response-content-disposition", disposition)
	}
	u, err := client.PresignedGetObject(ctx, cfg.S3.Bucket, objectKey, 6*time.Hour, params)
	if err != nil {
		return "", gerror.Wrap(err, "生成录制文件地址失败")
	}
	return u.String(), nil
}

func s3UploadEndpoint(cfg *model.RecordingConfig) string {
	if cfg == nil {
		return ""
	}
	if ep := strings.TrimSpace(cfg.S3.EgressEndpoint); ep != "" {
		return ep
	}
	return strings.TrimSpace(cfg.S3.Endpoint)
}

func newRecordingS3Client(cfg *model.RecordingConfig) (*minio.Client, error) {
	endpoint := strings.TrimSpace(cfg.S3.Endpoint)
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, gerror.Wrap(err, "解析 S3 endpoint 失败")
	}
	secure := u.Scheme == "https"
	host := u.Host
	if host == "" {
		host = endpoint
	}
	lookup := minio.BucketLookupAuto
	if cfg.S3.ForcePathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(host, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.S3.AccessKey, cfg.S3.SecretKey, ""),
		Secure:       secure,
		Region:       cfg.S3.Region,
		BucketLookup: lookup,
	})
	if err != nil {
		return nil, gerror.Wrap(err, "创建 S3 客户端失败")
	}
	return client, nil
}

func (s *sSysRecording) OpenForDownload(ctx context.Context, id int64) (rc io.ReadCloser, filename string, size int64, err error) {
	row, cfg, err := loadReadyRecording(ctx, id)
	if err != nil {
		return nil, "", 0, err
	}
	client, err := newRecordingS3Client(cfg)
	if err != nil {
		return nil, "", 0, err
	}
	obj, err := client.GetObject(ctx, cfg.S3.Bucket, strings.TrimSpace(row.ObjectKey), minio.GetObjectOptions{})
	if err != nil {
		return nil, "", 0, gerror.Wrap(err, "读取录制文件失败")
	}
	stat, statErr := obj.Stat()
	if statErr != nil {
		_ = obj.Close()
		return nil, "", 0, gerror.Wrap(statErr, "读取录制文件失败")
	}
	return obj, recordingFileName(row), stat.Size, nil
}

func (s *sSysRecording) OpenForPlay(ctx context.Context, id int64, rangeHeader string) (st *sysin.RecordingPlayStream, err error) {
	row, cfg, err := loadReadyRecording(ctx, id)
	if err != nil {
		return nil, err
	}
	client, err := newRecordingS3Client(cfg)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(row.ObjectKey)
	stat, err := client.StatObject(ctx, cfg.S3.Bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, gerror.Wrap(err, "读取录制文件失败")
	}
	if stat.Size <= 0 {
		return nil, gerror.New("录制文件尚未就绪")
	}
	start, end, partial, err := parseByteRange(rangeHeader, stat.Size)
	if err != nil {
		return nil, err
	}
	opts := minio.GetObjectOptions{}
	if partial && stat.Size > 0 {
		if setErr := opts.SetRange(start, end); setErr != nil {
			return nil, gerror.Wrap(setErr, "读取录制文件失败")
		}
	}
	obj, err := client.GetObject(ctx, cfg.S3.Bucket, key, opts)
	if err != nil {
		return nil, gerror.Wrap(err, "读取录制文件失败")
	}
	return &sysin.RecordingPlayStream{
		Body:     obj,
		Filename: recordingFileName(row),
		Size:     stat.Size,
		Start:    start,
		End:      end,
		Partial:  partial,
	}, nil
}

func loadReadyRecording(ctx context.Context, id int64) (*entity.Recording, *model.RecordingConfig, error) {
	user := contexts.GetUser(ctx)
	if user == nil || user.Id <= 0 {
		return nil, nil, gerror.New("请先登录")
	}
	if id <= 0 {
		return nil, nil, gerror.New("录制ID不能为空")
	}
	var row *entity.Recording
	if err := recordingModel(ctx).Where("id", id).Scan(&row); err != nil {
		return nil, nil, gerror.Wrap(err, "查询录制记录失败")
	}
	if row == nil || row.Id == 0 {
		return nil, nil, gerror.New("录制不存在")
	}
	if row.Status != consts.RecordingStatusComplete {
		return nil, nil, gerror.New("录制文件尚未就绪")
	}
	key := strings.TrimSpace(row.ObjectKey)
	if key == "" || strings.Contains(key, "{") {
		return nil, nil, gerror.New("录制文件不存在")
	}
	cfg, err := loadRecordingConfig(ctx)
	if err != nil {
		return nil, nil, err
	}
	return row, cfg, nil
}

func ensureRecordingBucket(ctx context.Context, cfg *model.RecordingConfig) error {
	bucketOnce.Do(func() {
		client, err := newRecordingS3Client(cfg)
		if err != nil {
			bucketErr = err
			return
		}
		exists, err := client.BucketExists(ctx, cfg.S3.Bucket)
		if err != nil {
			bucketErr = gerror.Wrap(err, "检查 bucket 失败")
			return
		}
		if !exists {
			if err = client.MakeBucket(ctx, cfg.S3.Bucket, minio.MakeBucketOptions{Region: cfg.S3.Region}); err != nil {
				bucketErr = gerror.Wrap(err, "创建 bucket 失败")
				return
			}
		}
	})
	return bucketErr
}
