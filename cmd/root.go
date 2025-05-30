package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"dockerops/internal/config"
	"dockerops/internal/puller"

	"github.com/spf13/cobra"
)

const VERSION = "v2.0.0"

var (
	configFile string
	image      string
	arch       string
	username   string
	password   string
	quiet      bool
	debug      bool
	prefix     string
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "DockerOps",
	Short: "增强版 Docker 镜像拉取工具 - 支持多仓库搜索和镜像管理",
	Long: `DockerOps 是一个增强版的 Docker 镜像拉取工具，支持：
- 多镜像仓库搜索和自动故障转移
- 配置文件管理镜像仓库（首次运行时自动创建）
- 标签转换规则
- 跨平台支持 (Windows, Linux, macOS)
- 进度条显示和并发下载
- Docker镜像推送、加载、保存等操作

首次使用时，工具会自动创建默认配置文件 config.json
您也可以使用 'DockerOps config init' 手动初始化配置文件`,
	Version: VERSION,
	Run:     runPull,
}

// pullCmd 拉取命令
var pullCmd = &cobra.Command{
	Use:   "pull [IMAGE]",
	Short: "拉取Docker镜像",
	Long:  "从配置的多个镜像仓库中搜索并拉取Docker镜像",
	Args:  cobra.MaximumNArgs(1),
	Run:   runPull,
}

// pushCmd 推送命令
var pushCmd = &cobra.Command{
	Use:   "push [PREFIX]",
	Short: "推送镜像到仓库",
	Long:  "将匹配指定前缀的镜像推送到Docker仓库",
	Args:  cobra.ExactArgs(1),
	Run:   runPush,
}

// loadCmd 加载命令
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "从本地tar文件加载镜像",
	Long:  "从当前目录的所有.tar文件中加载Docker镜像",
	Run:   runLoad,
}

// saveCmd 保存命令
var saveCmd = &cobra.Command{
	Use:   "save [PREFIX]",
	Short: "保存镜像到本地tar文件",
	Long:  "将匹配指定前缀的镜像保存为tar文件",
	Args:  cobra.ExactArgs(1),
	Run:   runSave,
}

// saveComposeCmd 保存compose镜像命令
var saveComposeCmd = &cobra.Command{
	Use:   "save-compose",
	Short: "保存docker-compose.yml中的镜像",
	Long:  "从docker-compose.yml文件中提取镜像并保存为tar文件",
	Run:   runSaveCompose,
}

// matchCmd 匹配命令
var matchCmd = &cobra.Command{
	Use:   "match [PREFIX]",
	Short: "匹配指定前缀的镜像",
	Long:  "列出所有匹配指定前缀的Docker镜像",
	Args:  cobra.ExactArgs(1),
	Run:   runMatch,
}

// listCmd 列出仓库命令
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "列出配置的镜像仓库",
	Long:  "显示配置文件中所有镜像仓库的信息",
	Run:   runList,
}

// configCmd 配置命令
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置管理",
	Long:  "管理配置文件和设置",
}

// configShowCmd 显示配置命令
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "显示当前配置",
	Long:  "显示当前配置文件的内容",
	Run:   runConfigShow,
}

// configInitCmd 初始化配置命令
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "初始化配置文件",
	Long:  "创建默认的配置文件，如果文件已存在则询问是否覆盖",
	Run:   runConfigInit,
}

// searchCmd 搜索命令
var searchCmd = &cobra.Command{
	Use:   "search [IMAGE]",
	Short: "搜索Docker镜像",
	Long:  "使用高级API搜索Docker镜像信息",
	Args:  cobra.ExactArgs(1),
	Run:   runSearch,
}

func init() {
	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.json", "配置文件路径")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "启用调试模式")

	// 添加拉取命令标志
	pullCmd.Flags().StringVarP(&image, "image", "i", "", "Docker 镜像名称（例如：nginx:latest）")
	pullCmd.Flags().StringVarP(&arch, "arch", "a", "", "架构，默认：amd64")
	pullCmd.Flags().StringVarP(&username, "username", "u", "", "Docker 仓库用户名")
	pullCmd.Flags().StringVarP(&password, "password", "p", "", "Docker 仓库密码")
	pullCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "静默模式，减少交互")

	// 添加搜索命令标志
	searchCmd.Flags().StringVarP(&arch, "arch", "a", "", "架构过滤，例如：amd64")

	// 添加子命令
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(loadCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(saveComposeCmd)
	rootCmd.AddCommand(matchCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
}

// showBanner 显示DockerOps的ASCII艺术图案
func showBanner() {
	fmt.Println(`
    ____             __              ____            
   / __ \____  _____/ /_____  _____ / __ \____  _____
  / / / / __ \/ ___/ //_/ _ \/ ___// / / / __ \/ ___/
 / /_/ / /_/ / /__/ ,< /  __/ /   / /_/ / /_/ (__  ) 
/_____/\____/\___/_/|_|\___/_/    \____/ .___/____/  
                                      /_/            `)
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "执行命令时出错: %v\n", err)
		os.Exit(1)
	}
}

// runPull 执行拉取命令
func runPull(cmd *cobra.Command, args []string) {
	// 显示帮助信息
	if len(args) == 0 && !quiet {
		showBanner()
		fmt.Println("\n这是一个多功能的 Docker 镜像管理工具，支持以下功能：")
		fmt.Println("  - pull: 拉取Docker镜像")
		fmt.Println("  - push: 推送镜像到仓库")
		fmt.Println("  - load: 从本地tar文件加载镜像")
		fmt.Println("  - save: 保存镜像到本地tar文件")
		fmt.Println("  - save-compose: 保存docker-compose.yml中的镜像")
		fmt.Println("  - match: 匹配指定前缀的镜像")
		fmt.Println("  - list: 列出配置的镜像仓库")
		fmt.Println("  - config show: 显示当前配置")
		fmt.Println("  - config init: 初始化配置文件")
		fmt.Println("\n使用 'DockerOps [command] --help' 查看具体命令帮助")
		fmt.Println("\n示例:")
		fmt.Println("  DockerOps pull nginx:latest")
		fmt.Println("  DockerOps pull nginx:latest --arch arm64")
		fmt.Println("  DockerOps list")
		fmt.Println("  DockerOps config show")
		fmt.Println("  DockerOps config init")
		fmt.Println("\n💡 首次使用时会自动创建配置文件，您也可以手动编辑 config.json 来自定义设置")
		return
	}

	// 设置日志级别
	if debug {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// 加载配置
	configManager := config.NewConfigManager(configFile)
	imagePuller := puller.NewMultiRegistryImagePuller(configManager)

	// 确保在程序结束时清理临时目录
	defer imagePuller.CleanupTmpDir()

	// 获取镜像名称
	if len(args) > 0 {
		image = args[0]
	}

	if image == "" {
		if !quiet {
			fmt.Print("请输入 Docker 镜像名称（例如：nginx:latest）：")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			image = strings.TrimSpace(input)
		}

		if image == "" {
			fmt.Fprintf(os.Stderr, "错误：镜像名称是必填项\n")
			os.Exit(1)
		}
	}

	// 显示个性化欢迎信息
	showBanner()
	fmt.Printf("正在为您拉取镜像: %s\n", image)

	// 获取架构
	if arch == "" {
		arch = configManager.GetConfig().Settings.DefaultArchitecture
		if !quiet {
			fmt.Printf("请输入架构（arm64/amd64，默认: %s）：", arch)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				if input != "arm64" && input != "amd64" {
					fmt.Fprintf(os.Stderr, "错误：架构只能是 arm64 或 amd64\n")
					os.Exit(1)
				}
				arch = input
			}
		}
	} else if arch != "arm64" && arch != "amd64" {
		fmt.Fprintf(os.Stderr, "错误：架构只能是 arm64 或 amd64\n")
		os.Exit(1)
	}

	// 获取认证信息
	if username == "" && !quiet {
		fmt.Print("请输入镜像仓库用户名（可选）：")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		username = strings.TrimSpace(input)
	}

	if password == "" && !quiet && username != "" {
		fmt.Print("请输入镜像仓库密码：")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		password = strings.TrimSpace(input)
	}

	// 拉取镜像
	outputFile, err := imagePuller.PullImage(image, arch, username, password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "拉取镜像失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n🎉 镜像拉取成功！输出文件：%s\n", outputFile)
}

// runPush 执行推送命令
func runPush(cmd *cobra.Command, args []string) {
	prefix := args[0]
	fmt.Printf("正在推送匹配前缀 '%s' 的镜像到仓库...\n", prefix)

	// 获取匹配的镜像
	images, err := getMatchingImages(prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取镜像列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Printf("未找到匹配前缀 '%s' 的镜像\n", prefix)
		return
	}

	// 推送每个镜像
	for _, image := range images {
		fmt.Printf("正在推送镜像: %s\n", image)
		if err := runDockerCommand("push", image); err != nil {
			fmt.Fprintf(os.Stderr, "推送镜像 %s 失败: %v\n", image, err)
		} else {
			fmt.Printf("✅ 成功推送镜像: %s\n", image)
		}
	}

	fmt.Println("推送操作完成！")
}

// runLoad 执行加载命令
func runLoad(cmd *cobra.Command, args []string) {
	fmt.Println("正在从本地tar文件加载镜像...")

	// 查找当前目录下的所有.tar文件
	tarFiles, err := filepath.Glob("*.tar")
	if err != nil {
		fmt.Fprintf(os.Stderr, "查找tar文件失败: %v\n", err)
		os.Exit(1)
	}

	if len(tarFiles) == 0 {
		fmt.Println("当前目录下未找到.tar文件")
		return
	}

	// 加载每个tar文件
	for _, tarFile := range tarFiles {
		fmt.Printf("正在加载镜像: %s\n", tarFile)
		if err := runDockerCommand("load", "-i", tarFile); err != nil {
			fmt.Fprintf(os.Stderr, "加载镜像 %s 失败: %v\n", tarFile, err)
		} else {
			fmt.Printf("✅ 成功加载镜像: %s\n", tarFile)
		}
	}

	fmt.Println("加载操作完成！")
}

// runSave 执行保存命令
func runSave(cmd *cobra.Command, args []string) {
	prefix := args[0]
	fmt.Printf("正在保存匹配前缀 '%s' 的镜像到本地tar文件...\n", prefix)

	// 获取匹配的镜像
	images, err := getMatchingImages(prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取镜像列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Printf("未找到匹配前缀 '%s' 的镜像\n", prefix)
		return
	}

	// 创建保存目录
	dirName := fmt.Sprintf("images_%s", strings.ReplaceAll(prefix, "/", "_"))
	if err := os.MkdirAll(dirName, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录失败: %v\n", err)
		os.Exit(1)
	}

	// 保存镜像列表
	listFile := filepath.Join(dirName, "list.txt")
	file, err := os.Create(listFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建列表文件失败: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	for _, image := range images {
		file.WriteString(image + "\n")
	}

	// 保存每个镜像
	for i, image := range images {
		parts := strings.Split(image, ":")
		imageName := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}

		// 生成文件名
		baseImageName := filepath.Base(imageName)
		tarFileName := fmt.Sprintf("%s-%s.tar", baseImageName, tag)
		tarFilePath := filepath.Join(dirName, tarFileName)

		fmt.Printf("正在保存镜像 %d/%d: %s\n", i+1, len(images), image)
		if err := runDockerCommand("save", "-o", tarFilePath, image); err != nil {
			fmt.Fprintf(os.Stderr, "保存镜像 %s 失败: %v\n", image, err)
		} else {
			fmt.Printf("✅ 成功保存镜像: %s -> %s\n", image, tarFileName)
		}
	}

	fmt.Printf("保存操作完成！文件保存在目录: %s\n", dirName)
}

// runSaveCompose 执行保存compose镜像命令
func runSaveCompose(cmd *cobra.Command, args []string) {
	fmt.Println("正在从docker-compose.yml文件中提取并保存镜像...")

	// 检查docker-compose.yml文件是否存在
	if _, err := os.Stat("docker-compose.yml"); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "当前目录下未找到docker-compose.yml文件\n")
		os.Exit(1)
	}

	// 创建images目录
	if err := os.MkdirAll("images", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建images目录失败: %v\n", err)
		os.Exit(1)
	}

	// 读取docker-compose.yml文件
	content, err := os.ReadFile("docker-compose.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取docker-compose.yml文件失败: %v\n", err)
		os.Exit(1)
	}

	// 提取镜像名称
	images := extractImagesFromCompose(string(content))
	if len(images) == 0 {
		fmt.Println("docker-compose.yml文件中未找到镜像定义")
		return
	}

	// 保存每个镜像
	for _, image := range images {
		parts := strings.Split(image, ":")
		imageName := parts[0]
		tag := "latest"
		if len(parts) > 1 {
			tag = parts[1]
		}

		// 生成文件名
		fileName := fmt.Sprintf("%s_%s.tar", strings.ReplaceAll(imageName, "/", "_"), tag)
		filePath := filepath.Join("images", fileName)

		fmt.Printf("正在保存镜像: %s 到 %s\n", image, fileName)
		if err := runDockerCommand("save", image, "-o", filePath); err != nil {
			fmt.Fprintf(os.Stderr, "保存镜像 %s 失败: %v\n", image, err)
		} else {
			fmt.Printf("✅ 镜像已保存: %s\n", fileName)
		}
		fmt.Println("------------------------")
	}

	fmt.Println("所有镜像已保存完毕。")
}

// runMatch 执行匹配命令
func runMatch(cmd *cobra.Command, args []string) {
	prefix := args[0]
	fmt.Printf("匹配前缀 '%s' 的镜像:\n", prefix)

	images, err := getMatchingImages(prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取镜像列表失败: %v\n", err)
		os.Exit(1)
	}

	if len(images) == 0 {
		fmt.Printf("未找到匹配前缀 '%s' 的镜像\n", prefix)
		return
	}

	for _, image := range images {
		fmt.Println(image)
	}
}

// runList 执行列出仓库命令
func runList(cmd *cobra.Command, args []string) {
	showBanner()
	configManager := config.NewConfigManager(configFile)
	registries := configManager.GetRegistries()

	fmt.Println("配置的镜像仓库:")
	fmt.Println("================")

	for i, registry := range registries {
		fmt.Printf("%d. %s\n", i+1, registry.Name)
		fmt.Printf("   URL: %s\n", registry.URL)
		fmt.Printf("   优先级: %d\n", registry.Priority)
		fmt.Printf("   需要认证: %t\n", registry.AuthRequired)
		fmt.Printf("   超时时间: %d秒\n", registry.Timeout)
		fmt.Printf("   描述: %s\n", registry.Description)
		fmt.Println()
	}
}

// runConfigShow 显示配置
func runConfigShow(cmd *cobra.Command, args []string) {
	showBanner()
	configManager := config.NewConfigManager(configFile)
	config := configManager.GetConfig()

	fmt.Println("当前配置:")
	fmt.Println("==========")

	fmt.Printf("配置文件: %s\n", configFile)
	fmt.Printf("镜像仓库数量: %d\n", len(config.Registries))
	fmt.Printf("标签转换: %t\n", config.TagTransform.Enabled)
	fmt.Printf("默认架构: %s\n", config.Settings.DefaultArchitecture)
	fmt.Printf("最大并发仓库: %d\n", config.Settings.MaxConcurrentRegistries)
	fmt.Printf("移除仓库前缀: %t\n", config.Settings.RemoveRegistryPrefix)
	fmt.Printf("启用进度条: %t\n", config.Settings.EnableProgressBar)
	fmt.Printf("清理临时文件: %t\n", config.Settings.CleanupTempFiles)

	fmt.Println("\n标签转换规则:")
	for i, rule := range config.TagTransform.Rules {
		status := "禁用"
		if rule.Enabled {
			status = "启用"
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, rule.Name, status)
		fmt.Printf("     模式: %s\n", rule.Pattern)
		fmt.Printf("     替换: %s\n", rule.Replacement)
		fmt.Printf("     描述: %s\n", rule.Description)
	}
}

// runConfigInit 执行配置初始化命令
func runConfigInit(cmd *cobra.Command, args []string) {
	showBanner()
	// 检查配置文件是否已存在
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("配置文件 %s 已存在\n", configFile)
		fmt.Print("是否要覆盖现有配置文件？(y/N): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response != "y" && response != "yes" {
			fmt.Println("操作已取消")
			return
		}
	}

	// 创建配置管理器并保存默认配置
	configManager := config.NewConfigManager(configFile)

	if err := configManager.SaveConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "创建配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 成功创建配置文件: %s\n", configFile)
	fmt.Println("💡 您可以编辑此文件来自定义镜像仓库配置")
	fmt.Printf("📖 使用 'DockerOps config show' 查看当前配置\n")
}

// 辅助函数

// getMatchingImages 获取匹配指定前缀的镜像列表
func getMatchingImages(prefix string) ([]string, error) {
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var matchingImages []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.Contains(line, prefix) {
			matchingImages = append(matchingImages, line)
		}
	}

	return matchingImages, nil
}

// runDockerCommand 执行Docker命令
func runDockerCommand(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// extractImagesFromCompose 从docker-compose.yml内容中提取镜像名称
func extractImagesFromCompose(content string) []string {
	var images []string

	// 使用正则表达式匹配image行
	re := regexp.MustCompile(`(?m)^\s*image:\s*["']?([^"'\s]+)["']?`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		if len(match) > 1 {
			image := strings.TrimSpace(match[1])
			if image != "" {
				images = append(images, image)
			}
		}
	}

	return images
}

// runSearch 执行搜索命令
func runSearch(cmd *cobra.Command, args []string) {
	showBanner()
	image := args[0]
	fmt.Printf("正在使用高级API搜索Docker镜像: %s\n", image)

	// 加载配置
	configManager := config.NewConfigManager(configFile)

	// 检查是否启用高级API
	if !configManager.GetConfig().Settings.EnableAdvancedAPI {
		fmt.Println("❌ 高级API已禁用，请在配置文件中启用 enable_advanced_api")
		os.Exit(1)
	}

	// 创建API客户端
	apiClient := puller.NewAdvancedAPIClient()
	if configManager.GetConfig().Settings.AdvancedAPIURL != "" {
		apiClient.SetBaseURL(configManager.GetConfig().Settings.AdvancedAPIURL)
	}

	// 获取架构
	arch := cmd.Flag("arch").Value.String()
	platform := ""
	if arch != "" {
		platform = fmt.Sprintf("linux/%s", arch)
	}

	// 执行搜索
	results, err := apiClient.SearchImage(image, "", platform)
	if err != nil {
		fmt.Fprintf(os.Stderr, "搜索镜像失败: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("未找到匹配的镜像")
		return
	}

	fmt.Printf("\n找到 %d 个匹配的镜像:\n", len(results))
	fmt.Println("=" + strings.Repeat("=", 80))

	for i, result := range results {
		fmt.Printf("\n[%d] %s\n", i+1, result.Source)
		fmt.Printf("    镜像源: %s\n", result.Mirror)
		fmt.Printf("    平台: %s\n", result.Platform)
		fmt.Printf("    大小: %s\n", result.Size)
		fmt.Printf("    创建时间: %s\n", result.CreatedAt)
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("提示: 使用 'dockerops pull %s' 来拉取镜像\n", image)
}

// searchDockerImage 执行Docker搜索命令（保留作为备用）
func searchDockerImage(image, arch string) ([]string, error) {
	cmd := exec.Command("docker", "search", "--format", "{{.Name}}")
	if arch != "" {
		cmd.Args = append(cmd.Args, "--filter", "is-official=true", "--filter", "arch="+arch)
	}
	cmd.Args = append(cmd.Args, image)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			results = append(results, line)
		}
	}

	return results, nil
}
