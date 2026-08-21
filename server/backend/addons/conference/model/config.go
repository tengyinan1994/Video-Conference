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
	Enabled        bool           `json:"enabled"`
	S3             RecordingS3    `json:"s3"`
	PublicEndpoint string         `json:"publicEndpoint"`
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
