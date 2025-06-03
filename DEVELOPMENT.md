 # DockerOps 开发文档

## Docker 镜像打包兼容性问题排查与解决

### 问题背景

在从 Python 版本迁移到 Go 版本的过程中，遇到了 `docker load` 导入镜像时的兼容性问题。源项目（Python版本）可以正常导入，但 Go 版本生成的 tar 包无法被 Docker 正确加载。

### 问题现象

#### 第一个错误：Config 文件路径错误
```bash
docker load -i ddn-k8s_docker.io_nginx_latest_amd64.tar 
open /iflytek/data/docker/tmp/docker-import-1504736865/tmp/e4fff0779e6ddd22366469f08626c3ab1884b5cbe1719b26da238c95f247b305.json: no such file or directory
```

#### 第二个错误：层文件路径错误
```bash
docker load -i ddn-k8s_docker.io_library_busybox_latest_amd64.tar
open /iflytek/data/docker/tmp/docker-import-1487202911/c267801a885425376839534d4d9795ddb0797c3caab4d8d86a2903b953209601/layer.tar: no such file or directory
```

### 问题分析

通过对比 Python 版本和 Go 版本的代码，发现了以下关键问题：

#### 1. Config 文件路径引用错误
- **Python 版本**：`config_filename = f'{config_digest[7:]}.json'` 直接使用配置文件的 digest
- **Go 版本**：错误地使用了 `layers[0].Digest` 而不是 `manifest.Config.Digest`

#### 2. 层 ID 生成方式错误
- **Python 版本**：直接使用层的真实 digest（去掉 `sha256:` 前缀）作为目录名
- **Go 版本**：生成假的哈希 ID，导致 Docker 找不到对应的层文件

#### 3. Tar 包路径分隔符问题
- **Windows 系统**：使用 `\` 作为路径分隔符
- **Docker 期望**：tar 包内使用 `/` 作为路径分隔符

### 解决方案

#### 1. 修复 Config 文件引用

**位置**：`internal/puller/puller.go` 的 `downloadLayers` 函数

**修改前**：
```go
"Config": filepath.Base(tmpDir) + "/" + layers[0].Digest[7:] + ".json",
```

**修改后**：
```go
"Config": manifest.Config.Digest[7:] + ".json",
```

#### 2. 修复层 ID 生成逻辑

**位置**：`internal/puller/puller.go` 第 703-705 行

**修改前**：
```go
// 生成假的层ID
hash := sha256.Sum256([]byte(parentID + "\n" + layer.Digest + "\n"))
fakeLayerID := fmt.Sprintf("%x", hash)
```

**修改后**：
```go
// 使用真实的层digest作为ID（去掉sha256:前缀）
fakeLayerID := layer.Digest[7:] // 去掉 "sha256:" 前缀
```

#### 3. 修复 repositories 文件

**添加变量**（第 699 行附近）：
```go
var parentID string
var layerPaths []string
var lastLayerID string // 保存最后一层的ID
```

**更新循环逻辑**（第 746 行附近）：
```go
layerPaths = append(layerPaths, fakeLayerID+"/layer.tar")
parentID = fakeLayerID
lastLayerID = fakeLayerID // 保存最后一层的ID
```

**修改 repositories 文件生成**（第 770 行附近）：
```go
repositories := map[string]map[string]string{
    imageInfo.Image: {
        imageInfo.Tag: lastLayerID, // 使用最后一层的ID
    },
}
```

#### 4. 修复路径分隔符问题

**位置**：`createImageTar` 函数中的 `filepath.Walk` 回调

**修改前**：
```go
header.Name = relPath
```

**修改后**：
```go
header.Name = filepath.ToSlash(relPath)
```

### 技术要点

#### Docker 镜像格式理解

1. **manifest.json**：
   - 描述镜像的元数据
   - 包含配置文件路径和层文件路径
   - Config 字段必须指向正确的配置文件

2. **repositories**：
   - 映射镜像标签到层 ID
   - 必须使用最后一层的 ID

3. **层目录结构**：
   - 目录名必须使用真实的 digest（去掉 `sha256:` 前缀）
   - 每个层目录包含 `json` 和 `layer.tar` 文件

#### 跨平台兼容性

1. **路径分隔符**：
   - Windows：`\`
   - Unix/Linux：`/`
   - Tar 包内部：必须使用 `/`

2. **文件权限**：
   - 确保生成的文件具有正确的权限

### 验证方法

#### 1. 编译测试
```bash
go build -o DockerOps.exe
```

#### 2. 功能测试
```bash
./DockerOps.exe pull busybox:latest
docker load -i docker.io_library_busybox_latest_amd64.tar
```

#### 3. 检查生成的文件结构
```
tmp/
├── manifest.json          # 镜像元数据
├── repositories          # 标签映射
├── <config-digest>.json  # 镜像配置
└── <layer-digest>/       # 层目录
    ├── json              # 层元数据
    └── layer.tar         # 层数据
```

#### 4. 验证 manifest.json 内容
```json
[
  {
    "Config": "sha256hash.json",
    "RepoTags": ["busybox:latest"],
    "Layers": [
      "layerhash1/layer.tar",
      "layerhash2/layer.tar"
    ]
  }
]
```

### 经验总结

#### 1. 严格遵循规范
- Docker 镜像格式有严格的规范
- 不能随意修改文件名和目录结构
- 必须使用真实的 digest 值

#### 2. 跨平台开发注意事项
- 路径分隔符差异
- 文件权限差异
- 字符编码差异

#### 3. 调试方法
- 对比已知正确的实现
- 逐步验证每个组件
- 使用工具检查 tar 包内容

#### 4. 测试策略
- 单元测试：验证每个函数的正确性
- 集成测试：验证整个流程
- 兼容性测试：在不同平台上测试

### 相关代码文件

- `internal/puller/puller.go`：主要的镜像处理逻辑
- `cmd/root.go`：命令行接口
- `config.json`：配置文件

### 参考资料

- [Docker Image Specification](https://github.com/moby/moby/blob/master/image/spec/v1.2.md)
- [OCI Image Format Specification](https://github.com/opencontainers/image-spec)
- [Docker Registry HTTP API V2](https://docs.docker.com/registry/spec/api/)