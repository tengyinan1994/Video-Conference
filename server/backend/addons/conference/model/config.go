package model

// LiveKitConfig LiveKit 配置
type LiveKitConfig struct {
	Url                  string `json:"url"`
	ApiKey               string `json:"apiKey"`
	ApiSecret            string `json:"apiSecret"`
	TokenTTL             int64  `json:"tokenTTL"`
	AllowAnonymousToken  bool   `json:"allowAnonymousToken"`
	RateLimitPerMinute   int    `json:"rateLimitPerMinute"`
}
