package model

// LiveKitConfig LiveKit 配置
type LiveKitConfig struct {
	Url                 string `json:"url"`
	ApiUrl              string `json:"apiUrl"` // 服务端 RoomService/Egress HTTP；空则从 Url 推导
	ApiKey              string `json:"apiKey"`
	ApiSecret           string `json:"apiSecret"`
	TokenTTL            int64  `json:"tokenTTL"`
	AllowAnonymousToken bool   `json:"allowAnonymousToken"`
	RateLimitPerMinute  int    `json:"rateLimitPerMinute"`
}

// RecordingConfig 录制 → RustFS
type RecordingConfig struct {
	Enabled        bool        `json:"enabled"`
	S3             RecordingS3 `json:"s3"`
	PublicEndpoint string      `json:"publicEndpoint"`
	// Egress 自定义布局页面 URL；空则用内置 speaker 布局
	CustomBaseUrl string `json:"customBaseUrl"`
	Width         int    `json:"width"`        // 合成宽，默认 2560（与会中投屏 2K 一致）
	Height        int    `json:"height"`       // 合成高，默认 1440
	Framerate     int    `json:"framerate"`    // 默认 60
	VideoBitrate  int    `json:"videoBitrate"` // kbps，默认 12000
}

type RecordingS3 struct {
	Endpoint       string `json:"endpoint"`       // HotGo 本机访问 RustFS
	EgressEndpoint string `json:"egressEndpoint"` // Egress 容器访问 RustFS；空则用 Endpoint
	AccessKey      string `json:"accessKey"`
	SecretKey      string `json:"secretKey"`
	Bucket         string `json:"bucket"`
	Region         string `json:"region"`
	ForcePathStyle bool   `json:"forcePathStyle"`
}

// MinutesConfig 会后 AI 纪要
type MinutesConfig struct {
	Enabled         bool   `json:"enabled"`
	WorkerUrl       string `json:"workerUrl"`       // Python Worker 基址，如 http://minutes-worker:8090
	CallbackSecret  string `json:"callbackSecret"`  // 派单/回调共享密钥
	MinTranscriptChars int `json:"minTranscriptChars"` // 低于此字数视为无有效发言
}
