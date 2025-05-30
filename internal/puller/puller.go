package puller

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dockerops/internal/config"

	"github.com/schollz/progressbar/v3"
)

// ImageInfo 镜像信息
type ImageInfo struct {
	Repository string
	Image      string
	Tag        string
}

// ManifestResponse 清单响应
type ManifestResponse struct {
	SchemaVersion int                `json:"schemaVersion"`
	MediaType     string             `json:"mediaType"`
	Config        ConfigDescriptor   `json:"config"`
	Layers        []LayerDescriptor  `json:"layers"`
	Manifests     []PlatformManifest `json:"manifests,omitempty"`
}

// ConfigDescriptor 配置描述符
type ConfigDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// LayerDescriptor 层描述符
type LayerDescriptor struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

// PlatformManifest 平台清单
type PlatformManifest struct {
	MediaType string   `json:"mediaType"`
	Size      int64    `json:"size"`
	Digest    string   `json:"digest"`
	Platform  Platform `json:"platform"`
}

// Platform 平台信息
type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

// AuthToken 认证令牌
type AuthToken struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// MultiRegistryImagePuller 多仓库镜像拉取器
type MultiRegistryImagePuller struct {
	configManager *config.ConfigManager
	registries    []config.RegistryConfig
	httpClient    *http.Client
	stopChan      chan struct{}
	apiClient     *AdvancedAPIClient // 添加高级API客户端
}

// NewMultiRegistryImagePuller 创建多仓库镜像拉取器
func NewMultiRegistryImagePuller(configManager *config.ConfigManager) *MultiRegistryImagePuller {
	// 创建HTTP客户端，禁用SSL验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	// 创建高级API客户端，使用配置中的URL
	apiClient := NewAdvancedAPIClient()
	if configManager.GetConfig().Settings.AdvancedAPIURL != "" {
		apiClient.baseURL = configManager.GetConfig().Settings.AdvancedAPIURL
	}

	return &MultiRegistryImagePuller{
		configManager: configManager,
		registries:    configManager.GetRegistries(),
		httpClient:    client,
		stopChan:      make(chan struct{}),
		apiClient:     apiClient,
	}
}

// ParseImageInput 解析镜像输入
func (p *MultiRegistryImagePuller) ParseImageInput(imageInput string) ImageInfo {
	// 检查是否包含私有仓库地址
	if strings.Contains(imageInput, "/") {
		parts := strings.Split(imageInput, "/")
		if len(parts) > 0 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
			// 私有仓库格式
			remainder := strings.Join(parts[1:], "/")
			repoParts := strings.Split(remainder, "/")

			var imgTag string
			if len(repoParts) == 1 {
				imgTag = repoParts[0]
			} else {
				imgTag = repoParts[len(repoParts)-1]
			}

			img, tag := parseImageTag(imgTag)
			repository := strings.Split(remainder, ":")[0]

			return ImageInfo{
				Repository: repository,
				Image:      img,
				Tag:        tag,
			}
		}
	}

	// 标准格式
	parts := strings.Split(imageInput, "/")
	var repo, imgTag string

	if len(parts) == 1 {
		// 对于单个名称的镜像，不自动添加library前缀
		// 用户需要明确指定是否为官方镜像
		repo = ""
		imgTag = parts[0]
	} else {
		repo = strings.Join(parts[:len(parts)-1], "/")
		imgTag = parts[len(parts)-1]
	}

	img, tag := parseImageTag(imgTag)

	var repository string
	if repo == "" {
		repository = img
	} else {
		repository = fmt.Sprintf("%s/%s", repo, img)
	}

	return ImageInfo{
		Repository: repository,
		Image:      img,
		Tag:        tag,
	}
}

// parseImageTag 解析镜像名和标签
func parseImageTag(imgTag string) (string, string) {
	parts := strings.Split(imgTag, ":")
	if len(parts) > 1 {
		return parts[0], parts[1]
	}
	return parts[0], "latest"
}

// TestRegistryAvailability 测试仓库可用性
func (p *MultiRegistryImagePuller) TestRegistryAvailability(registry *config.RegistryConfig) bool {
	start := time.Now()

	url := fmt.Sprintf("https://%s/v2/", registry.URL)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(registry.Timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("❌ %s 创建请求失败: %v", registry.Name, err)
		registry.Available = false
		return false
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ %s 连接失败: %v", registry.Name, err)
		registry.Available = false
		return false
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)
	registry.ResponseTime = &responseTime
	registry.Available = resp.StatusCode == 200 || resp.StatusCode == 401

	if registry.Available {
		log.Printf("✅ %s 可用 (响应时间: %.2fs)", registry.Name, responseTime.Seconds())
	} else {
		log.Printf("❌ %s 不可用 (状态码: %d)", registry.Name, resp.StatusCode)
	}

	return registry.Available
}

// GetAuthToken 获取认证令牌
func (p *MultiRegistryImagePuller) GetAuthToken(registry *config.RegistryConfig, repository, username, password string) (string, error) {
	url := fmt.Sprintf("https://%s/v2/", registry.URL)

	resp, err := p.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("获取认证信息失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return "", nil // 不需要认证
	}

	authHeader := resp.Header.Get("WWW-Authenticate")
	if authHeader == "" {
		return "", fmt.Errorf("未找到认证头")
	}

	// 解析认证头
	parts := strings.Split(authHeader, "\"")
	if len(parts) < 4 {
		return "", fmt.Errorf("认证头格式错误")
	}

	authURL := parts[1]
	service := parts[3]

	tokenURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", authURL, service, repository)

	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建认证请求失败: %v", err)
	}

	// 添加基本认证
	if username != "" && password != "" {
		auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	tokenResp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("获取令牌失败: %v", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != 200 {
		return "", fmt.Errorf("获取令牌失败，状态码: %d", tokenResp.StatusCode)
	}

	var authToken AuthToken
	if err := json.NewDecoder(tokenResp.Body).Decode(&authToken); err != nil {
		return "", fmt.Errorf("解析令牌失败: %v", err)
	}

	if authToken.Token != "" {
		return authToken.Token, nil
	}
	return authToken.AccessToken, nil
}

// FetchManifest 获取镜像清单
func (p *MultiRegistryImagePuller) FetchManifest(registry *config.RegistryConfig, repository, tag, token string) (*ManifestResponse, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry.URL, repository, tag)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取清单失败，状态码: %d", resp.StatusCode)
	}

	var manifest ManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("解析清单失败: %v", err)
	}

	return &manifest, nil
}

// SearchImageInRegistries 在多个仓库中搜索镜像
func (p *MultiRegistryImagePuller) SearchImageInRegistries(imageInput, arch, username, password string) (*config.RegistryConfig, *ManifestResponse, ImageInfo, error) {
	imageInfo := p.ParseImageInput(imageInput)

	// 应用标签转换规则
	transformedTag := p.configManager.TransformTag(imageInfo.Tag)
	imageInfo.Tag = transformedTag

	log.Printf("开始搜索镜像: %s:%s", imageInfo.Repository, imageInfo.Tag)

	// 检查是否启用高级API
	if p.configManager.GetConfig().Settings.EnableAdvancedAPI {
		// 首先尝试使用高级API搜索
		log.Printf("🚀 优先使用高级API搜索镜像...")

		// 构建搜索关键词
		searchTerm := fmt.Sprintf("%s:%s", imageInfo.Repository, imageInfo.Tag)

		// 转换架构格式
		platform := ""
		if arch != "" {
			platform = fmt.Sprintf("linux/%s", arch)
		}

		// 首先尝试精确搜索
		results, err := p.apiClient.SearchImage(searchTerm, "", platform)
		if err != nil || len(results) == 0 {
			// 如果没有结果，尝试只搜索镜像名（去掉标签）
			parts := strings.Split(searchTerm, ":")
			if len(parts) > 1 {
				results, err = p.apiClient.SearchImage(parts[0], "", platform)
			}
		}

		if err == nil && len(results) > 0 {
			// 选择最佳匹配
			var bestMatch *APIImageResult

			// 优先选择docker.io的镜像
			for i := range results {
				if strings.Contains(results[i].Source, "docker.io") {
					bestMatch = &results[i]
					break
				}
			}

			// 如果没有docker.io镜像，选择第一个
			if bestMatch == nil {
				bestMatch = &results[0]
			}

			if bestMatch != nil {
				log.Printf("✅ 高级API找到镜像: %s (大小: %s)", bestMatch.Source, bestMatch.Size)

				// 从API结果中提取仓库信息
				registryURL, imagePath := p.apiClient.ConvertToRegistryInfo(bestMatch)

				if registryURL != "" && imagePath != "" {
					// 创建临时仓库配置
					tempRegistry := &config.RegistryConfig{
						Name:         fmt.Sprintf("API-Mirror"),
						URL:          registryURL,
						Priority:     0, // 最高优先级
						AuthRequired: false,
						Timeout:      30,
						Description:  fmt.Sprintf("通过高级API发现的镜像仓库"),
						Available:    true,
					}

					// 测试仓库可用性
					if p.TestRegistryAvailability(tempRegistry) {
						// 创建临时imageInfo用于API仓库
						apiImageInfo := imageInfo
						apiImageInfo.Repository = imagePath

						// 获取认证令牌
						token, err := p.GetAuthToken(tempRegistry, apiImageInfo.Repository, username, password)
						if err != nil {
							log.Printf("⚠️ 无法获取API仓库的认证: %v，尝试无认证访问", err)
							token = ""
						}

						// 获取清单
						manifest, err := p.FetchManifest(tempRegistry, apiImageInfo.Repository, apiImageInfo.Tag, token)
						if err == nil {
							log.Printf("✅ 成功从高级API仓库获取镜像清单")

							// 处理多架构镜像
							if len(manifest.Manifests) > 0 {
								selectedDigest := p.selectManifest(manifest.Manifests, arch)
								if selectedDigest != "" {
									// 获取特定架构的清单
									archManifest, err := p.FetchManifestByDigest(tempRegistry, apiImageInfo.Repository, selectedDigest, token)
									if err == nil {
										manifest = archManifest
									}
								}
							}

							return tempRegistry, manifest, apiImageInfo, nil
						} else {
							log.Printf("⚠️ 从API仓库获取清单失败: %v", err)
						}
					} else {
						log.Printf("⚠️ API推荐的仓库不可用: %s", registryURL)
					}
				}
			}
		} else {
			log.Printf("⚠️ 高级API搜索失败或无结果: %v", err)
		}

		// 如果高级API失败，回退到传统的多仓库搜索
		log.Printf("🔄 回退到传统多仓库搜索...")
	} else {
		log.Printf("📋 高级API已禁用，使用传统多仓库搜索...")
	}

	// 重新解析原始镜像输入，确保使用正确的镜像信息进行传统搜索
	originalImageInfo := p.ParseImageInput(imageInput)
	originalImageInfo.Tag = p.configManager.TransformTag(originalImageInfo.Tag)

	log.Printf("开始在 %d 个仓库中搜索镜像: %s:%s", len(p.registries), originalImageInfo.Repository, originalImageInfo.Tag)

	// 测试仓库可用性
	var availableRegistries []config.RegistryConfig
	var wg sync.WaitGroup
	var mu sync.Mutex

	maxWorkers := p.configManager.GetConfig().Settings.MaxConcurrentRegistries
	semaphore := make(chan struct{}, maxWorkers)

	for _, registry := range p.registries {
		wg.Add(1)
		go func(reg config.RegistryConfig) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if p.TestRegistryAvailability(&reg) {
				mu.Lock()
				availableRegistries = append(availableRegistries, reg)
				mu.Unlock()
			}
		}(registry)
	}

	wg.Wait()

	if len(availableRegistries) == 0 {
		return nil, nil, originalImageInfo, fmt.Errorf("没有可用的镜像仓库")
	}

	// 按优先级和响应时间排序
	sort.Slice(availableRegistries, func(i, j int) bool {
		if availableRegistries[i].Priority != availableRegistries[j].Priority {
			return availableRegistries[i].Priority < availableRegistries[j].Priority
		}
		if availableRegistries[i].ResponseTime != nil && availableRegistries[j].ResponseTime != nil {
			return *availableRegistries[i].ResponseTime < *availableRegistries[j].ResponseTime
		}
		return false
	})

	log.Printf("发现 %d 个可用仓库", len(availableRegistries))

	// 依次尝试每个可用仓库
	for _, registry := range availableRegistries {
		log.Printf("正在尝试 %s (%s)...", registry.Name, registry.URL)

		// 使用原始的repository名称，不做任何修改
		// 用户需要明确指定完整的镜像路径
		searchRepository := originalImageInfo.Repository

		// 获取认证令牌
		token, err := p.GetAuthToken(&registry, searchRepository, username, password)
		if err != nil {
			log.Printf("无法获取 %s 的认证: %v", registry.Name, err)
			continue
		}

		// 获取清单
		manifest, err := p.FetchManifest(&registry, searchRepository, originalImageInfo.Tag, token)
		if err != nil {
			log.Printf("从 %s 获取清单失败: %v", registry.Name, err)
			continue
		}

		log.Printf("✅ 在 %s 找到镜像 %s:%s", registry.Name, originalImageInfo.Repository, originalImageInfo.Tag)

		// 处理多架构镜像
		if len(manifest.Manifests) > 0 {
			selectedDigest := p.selectManifest(manifest.Manifests, arch)
			if selectedDigest != "" {
				// 获取特定架构的清单
				archManifest, err := p.FetchManifestByDigest(&registry, searchRepository, selectedDigest, token)
				if err == nil {
					manifest = archManifest
				}
			}
		}

		// 更新imageInfo中的repository为实际搜索的repository
		originalImageInfo.Repository = searchRepository
		return &registry, manifest, originalImageInfo, nil
	}

	return nil, nil, originalImageInfo, fmt.Errorf("在所有可用仓库中都未找到镜像: %s:%s", originalImageInfo.Repository, originalImageInfo.Tag)
}

// selectManifest 选择适合指定架构的清单
func (p *MultiRegistryImagePuller) selectManifest(manifests []PlatformManifest, arch string) string {
	for _, m := range manifests {
		if m.Platform.Architecture == arch && m.Platform.OS == "linux" {
			return m.Digest
		}
	}
	return ""
}

// FetchManifestByDigest 通过digest获取清单
func (p *MultiRegistryImagePuller) FetchManifestByDigest(registry *config.RegistryConfig, repository, digest, token string) (*ManifestResponse, error) {
	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry.URL, repository, digest)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("获取清单失败，状态码: %d", resp.StatusCode)
	}

	var manifest ManifestResponse
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("解析清单失败: %v", err)
	}

	return &manifest, nil
}

// DownloadFileWithProgress 下载文件并显示进度
func (p *MultiRegistryImagePuller) DownloadFileWithProgress(url, token, savePath, desc string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 创建文件
	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	// 创建进度条
	var bar *progressbar.ProgressBar
	if p.configManager.GetConfig().Settings.EnableProgressBar {
		bar = progressbar.DefaultBytes(resp.ContentLength, desc)
	}

	// 复制数据
	var writer io.Writer = file
	if bar != nil {
		writer = io.MultiWriter(file, bar)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		os.Remove(savePath) // 删除部分下载的文件
		return fmt.Errorf("下载失败: %v", err)
	}

	return nil
}

// PullImage 拉取镜像
func (p *MultiRegistryImagePuller) PullImage(imageInput, arch, username, password string) (string, error) {
	// 搜索镜像
	registry, manifest, imageInfo, err := p.SearchImageInRegistries(imageInput, arch, username, password)
	if err != nil {
		return "", err
	}

	log.Printf("选择的仓库：%s (%s)", registry.Name, registry.URL)
	log.Printf("镜像：%s", imageInfo.Repository)
	log.Printf("标签：%s", imageInfo.Tag)
	log.Printf("架构：%s", arch)

	// 检查清单中的层
	if len(manifest.Layers) == 0 {
		return "", fmt.Errorf("清单中没有层")
	}

	// 获取认证令牌
	token, err := p.GetAuthToken(registry, imageInfo.Repository, username, password)
	if err != nil {
		return "", fmt.Errorf("获取认证失败: %v", err)
	}

	// 创建临时目录
	tmpDir := "tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	log.Println("开始下载")

	// 下载配置文件
	configFilename := manifest.Config.Digest[7:] + ".json"
	configPath := filepath.Join(tmpDir, configFilename)
	configURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry.URL, imageInfo.Repository, manifest.Config.Digest)

	if err := p.DownloadFileWithProgress(configURL, token, configPath, "Config"); err != nil {
		return "", fmt.Errorf("下载配置文件失败: %v", err)
	}

	// 下载层
	if err := p.downloadLayers(registry, imageInfo, manifest.Layers, token, tmpDir); err != nil {
		return "", fmt.Errorf("下载层失败: %v", err)
	}

	// 打包镜像
	outputFile, err := p.createImageTar(tmpDir, imageInfo, arch)
	if err != nil {
		return "", fmt.Errorf("打包镜像失败: %v", err)
	}

	log.Printf("✅ 镜像 %s:%s 下载完成！", imageInfo.Image, imageInfo.Tag)
	log.Printf("镜像已保存为: %s", outputFile)
	log.Printf("可使用以下命令导入镜像: docker load -i %s", outputFile)

	if p.configManager.GetConfig().Settings.RemoveRegistryPrefix {
		log.Printf("导入后的镜像标签: %s:%s", imageInfo.Image, imageInfo.Tag)
	} else {
		log.Printf("导入后的镜像标签: %s:%s", imageInfo.Repository, imageInfo.Tag)
	}

	return outputFile, nil
}

// downloadLayers 下载镜像层
func (p *MultiRegistryImagePuller) downloadLayers(registry *config.RegistryConfig, imageInfo ImageInfo, layers []LayerDescriptor, token, tmpDir string) error {
	// 创建层目录和JSON映射
	layerJSONMap := make(map[string]map[string]interface{})
	var parentID string
	var layerPaths []string

	// 下载所有层
	for i, layer := range layers {
		// 生成假的层ID
		hash := sha256.Sum256([]byte(parentID + "\n" + layer.Digest + "\n"))
		fakeLayerID := fmt.Sprintf("%x", hash)

		layerDir := filepath.Join(tmpDir, fakeLayerID)
		if err := os.MkdirAll(layerDir, 0755); err != nil {
			return fmt.Errorf("创建层目录失败: %v", err)
		}

		// 创建层JSON
		layerJSON := map[string]interface{}{
			"id": fakeLayerID,
		}
		if parentID != "" {
			layerJSON["parent"] = parentID
		}
		layerJSONMap[fakeLayerID] = layerJSON

		// 下载层文件
		layerURL := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry.URL, imageInfo.Repository, layer.Digest)
		gzipPath := filepath.Join(layerDir, "layer_gzip.tar")

		desc := fmt.Sprintf("Layer %d/%d", i+1, len(layers))
		if err := p.DownloadFileWithProgress(layerURL, token, gzipPath, desc); err != nil {
			return fmt.Errorf("下载层 %s 失败: %v", layer.Digest[:12], err)
		}

		// 解压层文件
		tarPath := filepath.Join(layerDir, "layer.tar")
		if err := p.decompressGzip(gzipPath, tarPath); err != nil {
			return fmt.Errorf("解压层失败: %v", err)
		}

		// 删除gzip文件
		os.Remove(gzipPath)

		// 写入层JSON
		jsonPath := filepath.Join(layerDir, "json")
		jsonData, _ := json.Marshal(layerJSON)
		if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
			return fmt.Errorf("写入层JSON失败: %v", err)
		}

		layerPaths = append(layerPaths, fakeLayerID+"/layer.tar")
		parentID = fakeLayerID
	}

	// 创建manifest.json
	repoTag := fmt.Sprintf("%s:%s", imageInfo.Image, imageInfo.Tag)
	if !p.configManager.GetConfig().Settings.RemoveRegistryPrefix {
		repoTag = fmt.Sprintf("%s:%s", imageInfo.Repository, imageInfo.Tag)
	}

	manifestContent := []map[string]interface{}{
		{
			"Config":   filepath.Base(tmpDir) + "/" + layers[0].Digest[7:] + ".json",
			"RepoTags": []string{repoTag},
			"Layers":   layerPaths,
		},
	}

	manifestData, _ := json.Marshal(manifestContent)
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("写入manifest.json失败: %v", err)
	}

	// 创建repositories文件
	repositories := map[string]map[string]string{
		imageInfo.Image: {
			imageInfo.Tag: parentID,
		},
	}

	repositoriesData, _ := json.Marshal(repositories)
	repositoriesPath := filepath.Join(tmpDir, "repositories")
	if err := os.WriteFile(repositoriesPath, repositoriesData, 0644); err != nil {
		return fmt.Errorf("写入repositories失败: %v", err)
	}

	return nil
}

// decompressGzip 解压gzip文件
func (p *MultiRegistryImagePuller) decompressGzip(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gzipReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, gzipReader)
	return err
}

// createImageTar 创建镜像tar文件
func (p *MultiRegistryImagePuller) createImageTar(tmpDir string, imageInfo ImageInfo, arch string) (string, error) {
	safeRepo := strings.ReplaceAll(imageInfo.Repository, "/", "_")
	outputFile := fmt.Sprintf("%s_%s_%s.tar", safeRepo, imageInfo.Tag, arch)

	file, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("创建tar文件失败: %v", err)
	}
	defer file.Close()

	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	// 遍历临时目录，添加所有文件到tar
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(tmpDir, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建tar头
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 如果是文件，写入内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("创建tar失败: %v", err)
	}

	return outputFile, nil
}

// CleanupTmpDir 清理临时目录
func (p *MultiRegistryImagePuller) CleanupTmpDir() {
	tmpDir := "tmp"
	if p.configManager.GetConfig().Settings.CleanupTempFiles {
		if err := os.RemoveAll(tmpDir); err != nil {
			log.Printf("清理临时目录失败: %v", err)
		} else {
			log.Println("临时目录已清理")
		}
	}
}
