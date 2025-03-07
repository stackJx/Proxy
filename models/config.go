package models

// Config 定义了配置文件的结构
type Config struct {
	ListenAddr   string   `json:"listen_addr"`    // 代理服务器监听的地址
	Targets      []Target `json:"targets"`        // 目标服务器列表
	Timeout      Timeout  `json:"timeout"`        // 超时配置
	MaxIdleConns int      `json:"max_idle_conns"` // 最大空闲连接数
	EnableLogs   bool     `json:"enable_logs"`    // 是否开启日志
}

// Target 定义了每个目标服务器的配置
type Target struct {
	Name       string `json:"name"`        // 目标服务器名称
	URL        string `json:"url"`         // 目标服务器地址（支持 HTTP/HTTPS）
	PathPrefix string `json:"path_prefix"` // 路径前缀，用于匹配请求转发
}

// Timeout 定义了超时设置（单位：秒）
type Timeout struct {
	ReadTimeout  int `json:"read_timeout"`  // 响应头超时时间
	WriteTimeout int `json:"write_timeout"` // 写超时时间
	IdleTimeout  int `json:"idle_timeout"`  // 空闲连接超时时间
}
