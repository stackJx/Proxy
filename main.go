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
	"sync"
	"time"

	"awesomeProject/handlers"
	"awesomeProject/models"
)

// ProxyHandler 反向代理处理器，包含所有目标服务器的代理对象和日志配置
type ProxyHandler struct {
	targets    map[string]*httputil.ReverseProxy // key 为路径前缀
	enableLogs bool                              // 是否显示日志
	logFile    *os.File                          // 日志文件句柄
	mu         sync.RWMutex                      // 用于保护配置更新
}

// responseWriter 用于包装 http.ResponseWriter 以捕获响应状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写 WriteHeader 方法捕获状态码
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

	// 生成日志文件名称，例如：logs/proxy_2025-03-07.log
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
func loadConfig(filename string) (*models.Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// newProxyHandler 根据配置创建反向代理处理器
func newProxyHandler(config *models.Config) (*ProxyHandler, error) {
	handler := &ProxyHandler{
		targets:    make(map[string]*httputil.ReverseProxy),
		enableLogs: config.EnableLogs,
	}

	// 创建自定义的 Transport，对 HTTPS 忽略证书验证
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:          config.MaxIdleConns,
		IdleConnTimeout:       time.Duration(config.Timeout.IdleTimeout) * time.Second,
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
			Director: func(req *http.Request) {
				originalURL := req.URL.String()
				req.URL.Scheme = localTargetURL.Scheme
				req.URL.Host = localTargetURL.Host
				req.Host = localTargetURL.Host
				// 记录 URL 重写信息
				if handler.enableLogs {
					log.Printf("[Director] 将 URL 从 %s 重写为 %s", originalURL, req.URL.String())
				}
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if handler.enableLogs {
					log.Printf("[ErrorHandler] 转发请求时出错，Path:%s, 错误: %v", r.URL.Path, err)
				}
				http.Error(w, "代理服务器错误", http.StatusBadGateway)
			},
		}

		handler.targets[target.PathPrefix] = proxy
		if handler.enableLogs {
			log.Printf("[Config] 已注册目标服务器：%s, 前缀：%s, 地址：%s",
				target.Name, target.PathPrefix, target.URL)
		}
	}

	return handler, nil
}

// ServeHTTP 处理所有进入的 HTTP 请求，根据路径前缀转发到对应的目标服务器
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if h.enableLogs {
		log.Printf("[请求开始] 方法: %s, 路径: %s, 远程地址: %s, 用户代理: %s",
			r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
	}

	h.mu.RLock()
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
	h.mu.RUnlock()

	// 未找到匹配的目标服务器，返回 404
	if matchedProxy == nil {
		if h.enableLogs {
			log.Printf("[404错误] 未匹配到目标服务器，路径: %s", r.URL.Path)
		}
		http.Error(w, "404 服务未找到", http.StatusNotFound)
		return
	}

	if h.enableLogs {
		log.Printf("[转发请求] 转发请求路径 [%s] 至前缀为 [%s] 的目标服务器",
			r.URL.Path, matchedPrefix)
	}

	// 包装 ResponseWriter 以捕获响应状态码
	wrapped := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// 调用反向代理处理请求
	matchedProxy.ServeHTTP(wrapped, r)

	// 请求处理完成后记录耗时
	if h.enableLogs {
		duration := time.Since(startTime)
		log.Printf("[请求结束] 路径: %s, 状态码: %d, 耗时: %v",
			r.URL.Path, wrapped.statusCode, duration)
	}
}

// UpdateConfig 更新代理处理器的配置
func (h *ProxyHandler) UpdateConfig(config *models.Config) error {
	newHandler, err := newProxyHandler(config)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.targets = newHandler.targets
	h.enableLogs = config.EnableLogs
	h.mu.Unlock()

	return nil
}

func main() {
	// 初始化日志
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("日志初始化失败: %v", err)
	}
	defer logFile.Close()

	// 配置文件路径
	configPath := "config.json"

	// 创建多路复用器
	mux := http.NewServeMux()

	// 从配置文件加载初始配置
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// 创建反向代理处理器
	proxyHandler, err := newProxyHandler(config)
	if err != nil {
		log.Fatal(err)
	}

	// 创建配置处理器，并传入更新回调函数
	configHandler := handlers.NewConfigHandler(configPath, func(newConfig *models.Config) error {
		return proxyHandler.UpdateConfig(newConfig)
	})

	// 添加配置API处理器
	mux.Handle("/api/config", configHandler)

	// 添加静态文件服务
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/", fs)

	// 创建管理服务器（用于配置界面）
	adminServer := &http.Server{
		Addr:    ":8080", // 管理界面端口
		Handler: mux,
		// 设置管理界面的超时参数
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 创建代理服务器
	proxyServer := &http.Server{
		Addr:    config.ListenAddr,
		Handler: proxyHandler,
		// 使用配置文件中的超时参数
		ReadTimeout:  time.Duration(config.Timeout.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Timeout.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.Timeout.IdleTimeout) * time.Second,
	}

	// 启动管理服务器（在后台）
	go func() {
		log.Printf("[管理服务器启动] 管理界面在 http://localhost:8080 启动")
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("管理服务器启动失败: %v", err)
		}
	}()

	// 启动代理服务器（主服务器）
	log.Printf("[代理服务器启动] 反向代理服务器正在 %s 启动", config.ListenAddr)
	if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("代理服务器启动失败: %v", err)
	}
}
