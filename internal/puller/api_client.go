package puller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIImageResult API搜索结果
type APIImageResult struct {
	Source    string `json:"source"`
	Mirror    string `json:"mirror"`
	Platform  string `json:"platform"`
	Size      string `json:"size"`
	CreatedAt string `json:"createdAt"`
}

// APIResponse API响应
type APIResponse struct {
	Count   int              `json:"count"`
	Error   bool             `json:"error"`
	Results []APIImageResult `json:"results"`
	Search  string           `json:"search"`
}

// AdvancedAPIClient 高级API客户端
type AdvancedAPIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAdvancedAPIClient 创建高级API客户端
func NewAdvancedAPIClient() *AdvancedAPIClient {
	return &AdvancedAPIClient{
		baseURL: "https://docker.aityp.com/api/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL 设置API基础URL
func (c *AdvancedAPIClient) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SearchImage 使用高级API搜索镜像
func (c *AdvancedAPIClient) SearchImage(imageName, site, platform string) ([]APIImageResult, error) {
	// 构建查询参数
	params := url.Values{}
	params.Add("search", imageName)

	if site != "" {
		params.Add("site", site)
	}

	if platform != "" {
		params.Add("platform", platform)
	}

	// 构建完整URL
	searchURL := fmt.Sprintf("%s/image?%s", c.baseURL, params.Encode())
	fmt.Printf("🔍 API请求URL: %s\n", searchURL)

	// 发送请求
	resp, err := c.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体用于调试
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	fmt.Printf("📡 API响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("📄 API响应内容: %s\n", string(body))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// 如果解析失败，尝试直接解析为数组
		var directResults []APIImageResult
		if err2 := json.Unmarshal(body, &directResults); err2 == nil {
			return directResults, nil
		}
		return nil, fmt.Errorf("解析API响应失败: %v, 原始响应: %s", err, string(body))
	}

	if apiResp.Error {
		return nil, fmt.Errorf("API返回错误: %s", apiResp.Search)
	}

	return apiResp.Results, nil
}

// GetBestMatch 获取最佳匹配的镜像
func (c *AdvancedAPIClient) GetBestMatch(imageName, arch string) (*APIImageResult, error) {
	// 首先尝试精确搜索
	results, err := c.SearchImage(imageName, "", "")
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		// 如果没有结果，尝试只搜索镜像名（去掉标签）
		parts := strings.Split(imageName, ":")
		if len(parts) > 1 {
			results, err = c.SearchImage(parts[0], "", "")
			if err != nil {
				return nil, err
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("未找到匹配的镜像")
	}

	// 选择最佳匹配 - 优先选择docker.io的镜像
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

	return bestMatch, nil
}

// ConvertToRegistryInfo 将API结果转换为仓库信息
func (c *AdvancedAPIClient) ConvertToRegistryInfo(apiResult *APIImageResult) (string, string) {
	// 使用mirror字段作为仓库地址
	mirrorURL := apiResult.Mirror
	if mirrorURL == "" {
		return "", ""
	}

	// 解析镜像地址
	parts := strings.Split(mirrorURL, "/")
	if len(parts) < 2 {
		return "", mirrorURL
	}

	// 第一部分是仓库地址
	registryURL := parts[0]

	// 其余部分是镜像路径
	imagePath := strings.Join(parts[1:], "/")

	return registryURL, imagePath
}
