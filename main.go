package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

// ProxyHandler 反向代理处理器，包含所有目标服务器的代理对象和日志配置
type ProxyHandler struct {
	targets    map[string]*httputil.ReverseProxy // key 为路径前缀
	enableLogs bool                              // 是否显示日志
	logFile    *os.File                          // 日志文件句柄
}

// responseWriter 用于包装 http.ResponseWriter 以捕获响应状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// 重写 WriteHeader 方法捕获状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// initLogger 初始化日志，将日志输出到控制台和文件中
func initLogger() (*os.File, error) {
	// 创建日志目录 logs
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 生成日志文件名称，例如：logs/proxy_2025-02-05.log
	currentTime := time.Now()
	logFileName := filepath.Join("logs", fmt.Sprintf("proxy_%s.log", currentTime.Format("2006-01-02")))

	// 打开或创建日志文件（追加模式）
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 创建多重输出，将日志同时写入标准输出和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	// 设置日志格式：日期、时间、短文件名
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return logFile, nil
}

// loadConfig 从指定的 JSON 文件中加载配置
func loadConfig(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// newProxyHandler 根据配置创建反向代理处理器
func newProxyHandler(config *Config) (*ProxyHandler, error) {
	handler := &ProxyHandler{
		targets:    make(map[string]*httputil.ReverseProxy),
		enableLogs: config.EnableLogs,
	}

	// 创建自定义的 Transport，对 HTTPS 忽略证书验证
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:    config.MaxIdleConns,
		IdleConnTimeout: time.Duration(config.Timeout.IdleTimeout) * time.Second,
		// 设置响应头超时时间
		ResponseHeaderTimeout: time.Duration(config.Timeout.ReadTimeout) * time.Second,
	}

	// 遍历配置中的每个目标服务器，生成对应的反向代理对象
	for _, target := range config.Targets {
		targetURL, err := url.Parse(target.URL)
		if err != nil {
			return nil, fmt.Errorf("解析目标 URL [%s] 失败: %v", target.URL, err)
		}
		// 为了在闭包中使用 targetURL，新建一个局部变量
		localTargetURL := targetURL

		proxy := &httputil.ReverseProxy{
			Transport: transport,
			// Director 修改请求，将其重写到目标服务器
			Director: func(req *http.Request) {
				originalURL := req.URL.String()
				req.URL.Scheme = localTargetURL.Scheme
				req.URL.Host = localTargetURL.Host
				req.Host = localTargetURL.Host
				// 记录 URL 重写信息
				log.Printf("[Director] 将 URL 从 %s 重写为 %s", originalURL, req.URL.String())
			},
			// 发生错误时返回 502 错误，并记录错误信息
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("[ErrorHandler] 转发请求时出错，Path:%s, 错误: %v", r.URL.Path, err)
				w.WriteHeader(http.StatusBadGateway)
				w.Write([]byte("代理服务器错误"))
			},
		}

		// 保证能够通过路径前缀匹配找到对应的代理
		handler.targets[target.PathPrefix] = proxy
		log.Printf("[Config] 已注册目标服务器：%s, 前缀：%s, 地址：%s", target.Name, target.PathPrefix, target.URL)
	}

	return handler, nil
}

// ServeHTTP 处理所有进入的 HTTP 请求，根据路径前缀转发到对应的目标服务器
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	log.Printf("[请求开始] 方法: %s, 路径: %s, 远程地址: %s, 用户代理: %s", r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())

	// 根据请求路径查找匹配的目标服务器（使用最长前缀匹配原则）
	var matchedPrefix string
	var matchedProxy *httputil.ReverseProxy
	for prefix, proxy := range h.targets {
		if strings.HasPrefix(r.URL.Path, prefix) {
			if len(prefix) > len(matchedPrefix) {
				matchedPrefix = prefix
				matchedProxy = proxy
			}
		}
	}

	// 未找到匹配的目标服务器，返回 404
	if matchedProxy == nil {
		log.Printf("[404错误] 未匹配到目标服务器，路径: %s", r.URL.Path)
		http.Error(w, "404 服务未找到", http.StatusNotFound)
		return
	}

	log.Printf("[转发请求] 转发请求路径 [%s] 至前缀为 [%s] 的目标服务器", r.URL.Path, matchedPrefix)

	// 包装 ResponseWriter 以捕获响应状态码
	wrapped := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// 调用反向代理处理请求
	matchedProxy.ServeHTTP(wrapped, r)

	// 请求处理完成后记录耗时
	duration := time.Since(startTime)
	log.Printf("[请求结束] 路径: %s, 状态码: %d, 耗时: %v", r.URL.Path, wrapped.statusCode, duration)
}

func main() {
	// 初始化日志，将日志输出到控制台和文件
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("日志初始化失败: %v", err)
	}
	// 程序退出时关闭日志文件
	defer logFile.Close()

	// 从 config.json 加载配置
	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// 创建反向代理处理器
	handler, err := newProxyHandler(config)
	if err != nil {
		log.Fatal(err)
	}

	// 创建 HTTP 服务器，并设置超时参数
	server := &http.Server{
		Addr:         config.ListenAddr,
		Handler:      handler,
		ReadTimeout:  time.Duration(config.Timeout.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Timeout.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.Timeout.IdleTimeout) * time.Second,
	}

	log.Printf("[服务器启动] 反向代理服务器正在 %s 启动", config.ListenAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
