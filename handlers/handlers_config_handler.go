package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"awesomeProject/models"
)

// ConfigHandler 处理配置相关的HTTP请求
type ConfigHandler struct {
	configPath     string
	mutex          sync.RWMutex
	updateCallback func(*models.Config) error
	lastModified   time.Time
}

// NewConfigHandler 创建一个新的配置处理器
func NewConfigHandler(configPath string, updateCallback func(*models.Config) error) *ConfigHandler {
	return &ConfigHandler{
		configPath:     configPath,
		updateCallback: updateCallback,
	}
}

// ServeHTTP 实现 http.Handler 接口
func (h *ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGet(w, r)
	case http.MethodPost:
		h.handlePost(w, r)
	default:
		http.Error(w, "方法不支持", http.StatusMethodNotAllowed)
	}
}

// handleGet 处理获取配置的请求
func (h *ConfigHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	data, err := ioutil.ReadFile(h.configPath)
	if err != nil {
		log.Printf("读取配置文件失败: %v", err)
		http.Error(w, "读取配置失败", http.StatusInternalServerError)
		return
	}

	var config models.Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("解析配置文件失败: %v", err)
		http.Error(w, "配置文件格式无效", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Last-Modified", h.lastModified.UTC().Format(http.TimeFormat))
	w.Write(data)
}

// handlePost 处理更新配置的请求
func (h *ConfigHandler) handlePost(w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("读取请求体失败: %v", err)
		http.Error(w, "读取请求失败", http.StatusBadRequest)
		return
	}

	var newConfig models.Config
	if err := json.Unmarshal(body, &newConfig); err != nil {
		log.Printf("解析请求体失败: %v", err)
		http.Error(w, "无效的JSON格式", http.StatusBadRequest)
		return
	}

	if err := validateConfig(&newConfig); err != nil {
		log.Printf("配置验证失败: %v", err)
		http.Error(w, fmt.Sprintf("配置验证失败: %v", err), http.StatusBadRequest)
		return
	}

	formattedJSON, err := json.MarshalIndent(newConfig, "", "    ")
	if err != nil {
		log.Printf("格式化JSON失败: %v", err)
		http.Error(w, "处理配置失败", http.StatusInternalServerError)
		return
	}

	if err := ioutil.WriteFile(h.configPath, formattedJSON, 0644); err != nil {
		log.Printf("写入配置文件失败: %v", err)
		http.Error(w, "保存配置失败", http.StatusInternalServerError)
		return
	}

	h.lastModified = time.Now()

	if h.updateCallback != nil {
		if err := h.updateCallback(&newConfig); err != nil {
			log.Printf("更新运行时配置失败: %v", err)
			http.Error(w, "配置已保存但更新运行时配置失败", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("配置已更新"))
}

// validateConfig 验证配置是否有效
func validateConfig(config *models.Config) error {
	if config.ListenAddr == "" {
		return fmt.Errorf("监听地址不能为空")
	}

	if config.MaxIdleConns < 0 {
		return fmt.Errorf("最大空闲连接数不能为负数")
	}

	if config.Timeout.ReadTimeout < 0 ||
		config.Timeout.WriteTimeout < 0 ||
		config.Timeout.IdleTimeout < 0 {
		return fmt.Errorf("超时时间不能为负数")
	}

	usedPrefixes := make(map[string]bool)
	for i, target := range config.Targets {
		if target.Name == "" {
			return fmt.Errorf("目标服务器 #%d 的名称不能为空", i+1)
		}
		if target.URL == "" {
			return fmt.Errorf("目标服务器 #%d 的URL不能为空", i+1)
		}
		if target.PathPrefix == "" {
			return fmt.Errorf("目标服务器 #%d 的路径前缀不能为空", i+1)
		}
		if usedPrefixes[target.PathPrefix] {
			return fmt.Errorf("目标服务器 #%d 的路径前缀 '%s' 重复", i+1, target.PathPrefix)
		}
		usedPrefixes[target.PathPrefix] = true
	}

	return nil
}
