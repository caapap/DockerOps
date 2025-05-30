package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RegistryConfig 镜像仓库配置
type RegistryConfig struct {
	Name         string         `json:"name"`
	URL          string         `json:"url"`
	Priority     int            `json:"priority"`
	AuthRequired bool           `json:"auth_required"`
	Timeout      int            `json:"timeout"`
	Description  string         `json:"description"`
	Available    bool           `json:"-"`
	ResponseTime *time.Duration `json:"-"`
}

// TagTransformRule 标签转换规则
type TagTransformRule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	Replacement string `json:"replacement"`
	Enabled     bool   `json:"enabled"`
}

// TagTransform 标签转换配置
type TagTransform struct {
	Enabled bool               `json:"enabled"`
	Rules   []TagTransformRule `json:"rules"`
}

// Settings 全局设置
type Settings struct {
	MaxConcurrentRegistries int    `json:"max_concurrent_registries"`
	RetryCount              int    `json:"retry_count"`
	RemoveRegistryPrefix    bool   `json:"remove_registry_prefix"`
	DefaultArchitecture     string `json:"default_architecture"`
	DownloadTimeout         int    `json:"download_timeout"`
	EnableProgressBar       bool   `json:"enable_progress_bar"`
	CleanupTempFiles        bool   `json:"cleanup_temp_files"`
	EnableAdvancedAPI       bool   `json:"enable_advanced_api"`
	AdvancedAPIURL          string `json:"advanced_api_url"`
}

// Config 主配置结构
type Config struct {
	Registries   []RegistryConfig `json:"registries"`
	TagTransform TagTransform     `json:"tag_transform"`
	Settings     Settings         `json:"settings"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	configFile string
	config     *Config
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configFile string) *ConfigManager {
	cm := &ConfigManager{
		configFile: configFile,
	}
	cm.loadConfig()
	return cm
}

// loadConfig 加载配置文件
func (cm *ConfigManager) loadConfig() {
	if _, err := os.Stat(cm.configFile); os.IsNotExist(err) {
		log.Printf("配置文件 %s 不存在，创建默认配置文件", cm.configFile)
		cm.config = cm.getDefaultConfig()

		// 自动创建默认配置文件
		if err := cm.SaveConfig(); err != nil {
			log.Printf("⚠️ 创建默认配置文件失败: %v，将使用内存中的默认配置", err)
		} else {
			log.Printf("✅ 已创建默认配置文件: %s", cm.configFile)
			log.Printf("💡 您可以编辑此文件来自定义镜像仓库配置")
		}
		return
	}

	data, err := os.ReadFile(cm.configFile)
	if err != nil {
		log.Printf("读取配置文件失败: %v，使用默认配置", err)
		cm.config = cm.getDefaultConfig()
		return
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("解析配置文件失败: %v，使用默认配置", err)
		cm.config = cm.getDefaultConfig()
		return
	}

	cm.config = &config
	log.Printf("已加载配置文件: %s", cm.configFile)
}

// getDefaultConfig 获取默认配置
func (cm *ConfigManager) getDefaultConfig() *Config {
	return &Config{
		Registries: []RegistryConfig{
			{
				Name:         "阿里云",
				URL:          "registry.cn-hangzhou.aliyuncs.com",
				Priority:     1,
				AuthRequired: false,
				Timeout:      15,
				Description:  "阿里云容器镜像服务",
			},
			{
				Name:         "腾讯云",
				URL:          "ccr.ccs.tencentyun.com",
				Priority:     2,
				AuthRequired: false,
				Timeout:      15,
				Description:  "腾讯云容器镜像服务",
			},
			{
				Name:         "华为云",
				URL:          "swr.cn-north-4.myhuaweicloud.com",
				Priority:     3,
				AuthRequired: false,
				Timeout:      15,
				Description:  "华为云容器镜像服务",
			},
			{
				Name:         "网易云",
				URL:          "hub.c.163.com",
				Priority:     4,
				AuthRequired: false,
				Timeout:      15,
				Description:  "网易云容器镜像服务",
			},
			{
				Name:         "Docker Hub",
				URL:          "registry-1.docker.io",
				Priority:     10,
				AuthRequired: false,
				Timeout:      30,
				Description:  "官方Docker Hub仓库（备用）",
			},
		},
		TagTransform: TagTransform{
			Enabled: true,
			Rules: []TagTransformRule{
				{
					Name:        "默认规则",
					Description: "保持原始标签不变",
					Pattern:     ".*",
					Replacement: "{original_tag}",
					Enabled:     true,
				},
			},
		},
		Settings: Settings{
			MaxConcurrentRegistries: 5,
			RetryCount:              3,
			RemoveRegistryPrefix:    true,
			DefaultArchitecture:     "amd64",
			DownloadTimeout:         300,
			EnableProgressBar:       true,
			CleanupTempFiles:        true,
			EnableAdvancedAPI:       true,
			AdvancedAPIURL:          "https://docker.aityp.com/api/v1",
		},
	}
}

// GetRegistries 获取排序后的仓库列表
func (cm *ConfigManager) GetRegistries() []RegistryConfig {
	registries := make([]RegistryConfig, len(cm.config.Registries))
	copy(registries, cm.config.Registries)

	// 按优先级排序
	sort.Slice(registries, func(i, j int) bool {
		return registries[i].Priority < registries[j].Priority
	})

	return registries
}

// TransformTag 根据配置规则转换标签
func (cm *ConfigManager) TransformTag(originalTag string) string {
	if !cm.config.TagTransform.Enabled {
		return originalTag
	}

	for _, rule := range cm.config.TagTransform.Rules {
		if !rule.Enabled {
			continue
		}

		matched, err := regexp.MatchString(rule.Pattern, originalTag)
		if err != nil {
			log.Printf("正则表达式错误: %v", err)
			continue
		}

		if matched {
			transformedTag := strings.ReplaceAll(rule.Replacement, "{original_tag}", originalTag)
			if transformedTag != originalTag {
				log.Printf("标签转换: %s -> %s", originalTag, transformedTag)
			}
			return transformedTag
		}
	}

	return originalTag
}

// GetConfig 获取配置
func (cm *ConfigManager) GetConfig() *Config {
	return cm.config
}

// SaveConfig 保存配置到文件
func (cm *ConfigManager) SaveConfig() error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(cm.configFile, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	log.Printf("配置已保存到: %s", cm.configFile)
	return nil
}

// ConfigFileExists 检查配置文件是否存在
func (cm *ConfigManager) ConfigFileExists() bool {
	_, err := os.Stat(cm.configFile)
	return !os.IsNotExist(err)
}

// CreateDefaultConfigFile 创建默认配置文件
func (cm *ConfigManager) CreateDefaultConfigFile() error {
	if cm.ConfigFileExists() {
		return fmt.Errorf("配置文件 %s 已存在", cm.configFile)
	}

	cm.config = cm.getDefaultConfig()
	return cm.SaveConfig()
}

// GetConfigFilePath 获取配置文件路径
func (cm *ConfigManager) GetConfigFilePath() string {
	return cm.configFile
}
