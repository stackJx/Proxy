package main

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
)

// Timeout 配置超时的结构体
type Timeout struct {
	ReadTimeout  int `json:"read_timeout"`  // 读取超时秒数
	WriteTimeout int `json:"write_timeout"` // 写入超时秒数
	IdleTimeout  int `json:"idle_timeout"`  // 空闲超时秒数
}

// Target 表示一个后端目标服务器的配置
type Target struct {
	Name       string `json:"name"`        // 后端服务名称
	URL        string `json:"url"`         // 后端服务 URL
	PathPrefix string `json:"path_prefix"` // 路由匹配使用的路径前缀
}

// Config 定义配置文件的结构体
type Config struct {
	ListenAddr   string   `json:"listen_addr"`    // 代理服务器监听地址，如 ":8080"
	EnableLogs   bool     `json:"enable_logs"`    // 是否启用日志记录
	MaxIdleConns int      `json:"max_idle_conns"` // 最大空闲连接数
	Timeout      Timeout  `json:"timeout"`        // 超时设置
	Targets      []Target `json:"targets"`        // 后端服务配置列表
}

var configPath = "config.json"

// 使用 embed 将 static 文件夹中的所有静态文件打包到二进制文件中
//
//go:embed static/*
var staticFiles embed.FS

// ProxyHandler 实现反向代理处理器
type ProxyHandler struct {
	targets    map[string]*httputil.ReverseProxy // 以路径前缀为键的反向代理映射
	enableLogs bool                              // 是否启用日志
	mu         sync.RWMutex                      // 更新配置时的互斥锁
}

// responseWriter 用于包装 http.ResponseWriter，从而捕获响应状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader 重写 WriteHeader 方法以记录状态码
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// initLogger 初始化日志系统：创建 logs 目录并配置日志输出到控制台和文件
func initLogger() (*os.File, error) {
	// 创建 logs 目录
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, fmt.Errorf("创建 logs 目录失败: %v", err)
	}
	// 根据当前日期生成日志文件名，例如 proxy_2025-03-07.log
	currentTime := time.Now()
	logFileName := filepath.Join("logs", fmt.Sprintf("proxy_%s.log", currentTime.Format("2006-01-02")))
	// 以追加模式打开或创建日志文件
	logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}
	// 将日志输出到标准输出和文件
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	// 设置日志格式：日期、时间、短文件名
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	return logFile, nil
}

// loadConfig 从指定的 JSON 配置文件中加载配置
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

// newProxyHandler 根据配置创建一个新的反向代理处理器
func newProxyHandler(config *Config) (*ProxyHandler, error) {
	handler := &ProxyHandler{
		targets:    make(map[string]*httputil.ReverseProxy),
		enableLogs: config.EnableLogs,
	}
	// 使用自定义 Transport，跳过 TLS 证书验证
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          config.MaxIdleConns,
		IdleConnTimeout:       time.Duration(config.Timeout.IdleTimeout) * time.Second,
		ResponseHeaderTimeout: time.Duration(config.Timeout.ReadTimeout) * time.Second,
	}
	// 遍历每个目标服务器配置，设置对应的反向代理
	for _, target := range config.Targets {
		targetURL, err := url.Parse(target.URL)
		if err != nil {
			return nil, fmt.Errorf("解析目标 URL [%s] 失败: %v", target.URL, err)
		}
		// 为闭包创建本地变量
		localTargetURL := targetURL
		proxy := &httputil.ReverseProxy{
			Transport: transport,
			// Director 用于修改请求，使其指向目标服务器
			Director: func(req *http.Request) {
				originalURL := req.URL.String()
				req.URL.Scheme = localTargetURL.Scheme
				req.URL.Host = localTargetURL.Host
				req.Host = localTargetURL.Host

				// 转发前移除路径前缀
				if strings.HasPrefix(req.URL.Path, target.PathPrefix) {
					req.URL.Path = strings.TrimPrefix(req.URL.Path, target.PathPrefix)
					// 确保修剪后的路径以 / 开头
					if req.URL.Path == "" || req.URL.Path[0] != '/' {
						req.URL.Path = "/" + req.URL.Path
					}
				}

				if handler.enableLogs {
					log.Printf("[请求转发] 将 URL 从 %s 重写为 %s", originalURL, req.URL.String())
				}
			},
			// ErrorHandler 在转发请求出错时调用
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				if handler.enableLogs {
					log.Printf("[错误处理] 请求 %s 转发失败: %v", r.URL.Path, err)
				}
				http.Error(w, "反向代理错误", http.StatusBadGateway)
			},
		}
		handler.targets[target.PathPrefix] = proxy
		if handler.enableLogs {
			log.Printf("[配置] 注册目标服务器：%s, 路径前缀：%s, URL: %s", target.Name, target.PathPrefix, target.URL)
		}
	}
	return handler, nil
}

// ServeHTTP 根据匹配到的路径前缀将请求转发到对应的后端服务器
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	if h.enableLogs {
		log.Printf("[请求开始] 方法: %s, 路径: %s, 远程地址: %s, 用户代理: %s",
			r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
	}
	// 查找匹配的目标服务器 (使用最长前缀匹配)
	h.mu.RLock()
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
	if matchedProxy == nil {
		if h.enableLogs {
			log.Printf("[404] 无匹配目标服务器，路径: %s", r.URL.Path)
		}
		http.Error(w, "404 未找到服务", http.StatusNotFound)
		return
	}
	if h.enableLogs {
		log.Printf("[转发请求] 将请求 [%s] 转发到路径前缀为 [%s] 的目标", r.URL.Path, matchedPrefix)
	}
	wrapped := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
	matchedProxy.ServeHTTP(wrapped, r)
	duration := time.Since(startTime)
	if h.enableLogs {
		log.Printf("[请求结束] 路径: %s, 状态码: %d, 耗时: %v", r.URL.Path, wrapped.statusCode, duration)
	}
}

// UpdateConfig 更新运行时的反向代理配置
func (h *ProxyHandler) UpdateConfig(config *Config) error {
	newHandler, err := newProxyHandler(config)
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.targets = newHandler.targets
	h.enableLogs = config.EnableLogs
	return nil
}

func main() {
	// 初始化日志系统
	logFile, err := initLogger()
	if err != nil {
		log.Fatalf("日志初始化失败: %v", err)
	}
	defer logFile.Close()

	// 从配置文件中加载配置
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建反向代理处理器
	proxyHandler, err := newProxyHandler(config)
	if err != nil {
		log.Fatalf("创建代理处理器失败: %v", err)
	}

	// 设置管理界面的静态文件服务，从嵌入的静态文件中获取
	contentFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("获取嵌入静态文件子目录失败: %v", err)
	}
	adminFileServer := http.FileServer(http.FS(contentFS))

	// 配置 HTTP 路由
	mux := http.NewServeMux()
	// 提供管理界面
	mux.Handle("/", adminFileServer)
	// 配置管理 API，路径为 /api/config
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// 读取配置文件返回 JSON 数据
			data, err := ioutil.ReadFile(configPath)
			if err != nil {
				http.Error(w, "读取配置失败", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
		case http.MethodPost:
			// 读取请求体，将新配置解析后保存到配置文件
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "读取请求失败", http.StatusBadRequest)
				return
			}
			var newConfig Config
			if err := json.Unmarshal(body, &newConfig); err != nil {
				http.Error(w, "无效的 JSON 数据", http.StatusBadRequest)
				return
			}
			formatted, err := json.MarshalIndent(newConfig, "", "    ")
			if err != nil {
				http.Error(w, "格式化配置失败", http.StatusInternalServerError)
				return
			}
			if err := ioutil.WriteFile(configPath, formatted, 0644); err != nil {
				http.Error(w, "写入配置文件失败", http.StatusInternalServerError)
				return
			}
			if err := proxyHandler.UpdateConfig(&newConfig); err != nil {
				http.Error(w, "更新运行时配置失败", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("配置更新成功"))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// 启动管理界面服务器，监听 8080 端口
	adminServer := &http.Server{
		Addr:         ":39456",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 启动代理服务器，监听配置文件中指定的地址
	proxyServer := &http.Server{
		Addr:         config.ListenAddr,
		Handler:      proxyHandler,
		ReadTimeout:  time.Duration(config.Timeout.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(config.Timeout.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(config.Timeout.IdleTimeout) * time.Second,
	}

	// 后台启动管理服务器
	go func() {
		log.Printf("管理服务器启动于 http://localhost:39456")
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("管理服务器错误: %v", err)
		}
	}()

	log.Printf("代理服务器启动于 %s", config.ListenAddr)
	if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("代理服务器错误: %v", err)
	}
}
