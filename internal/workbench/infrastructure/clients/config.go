package clients

// SettingConfig 数据源配置
type SettingConfig struct {
	TushareToken    string
	CrawlTimeOut    int64
	UpdateOnStart   bool
	RefreshInterval int64
}

// DefaultConfig 默认配置
func DefaultConfig() *SettingConfig {
	return &SettingConfig{
		CrawlTimeOut:    30,
		RefreshInterval: 5,
	}
}
