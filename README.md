# DockerOps

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v2.0.0-orange.svg)](https://github.com/yourusername/DockerOps/releases)

[English](./README.md) | [中文](./README_CN.md)

DockerOps is an enhanced Docker image pulling tool designed to solve Docker image pulling difficulties in China. It supports multi-registry search, automatic failover, concurrent downloads, and more.

## 🎬 Demo

![DockerOps Demo](test-speed.png)

*DockerOps successfully pulling a large image (vllm-openai:v0.7.2, 16.53GB) with real-time progress display and high-speed downloads*

## ✨ Features

- 🚀 **Multi-Registry Support** - Supports multiple Chinese registries including Alibaba Cloud, Tencent Cloud, Huawei Cloud
- 🔄 **Automatic Failover** - Automatically switches to the next registry when one is unavailable
- ⚡ **Concurrent Downloads** - Supports multi-threaded concurrent downloads for improved speed
- 📊 **Progress Bar Display** - Real-time download progress visualization
- 🔍 **Advanced Image Search** - Search images across multiple registries using advanced API with detailed information
- 🔧 **Configuration Management** - Manage image registries through JSON configuration files
- 🏷️ **Tag Transformation Rules** - Intelligent tag conversion and mapping
- 🌐 **Cross-Platform Support** - Supports Windows, Linux, macOS
- 📦 **Image Management** - Support for image push, load, save operations

## 🛠️ Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/caapap/DockerOps.git
cd DockerOps

# Build
go build -o DockerOps main.go

# Or use build scripts
# Windows
./build.bat

# Linux/macOS
./build.sh
```

### Pre-compiled Binaries

Download the corresponding platform's pre-compiled binary from the [Releases](https://github.com/caapap/DockerOps/releases) page.

## 📖 Usage

### Basic Usage

```bash
# Pull an image
./DockerOps pull nginx:latest

# Search for images using advanced API
./DockerOps search nginx

# Search with architecture filter
./DockerOps search --arch amd64 nginx

# Specify architecture
./DockerOps pull --arch linux/amd64 nginx:latest

# Quiet mode
./DockerOps pull --quiet nginx:latest

# Debug mode
./DockerOps pull --debug nginx:latest
```

### Advanced Usage

```bash
# Use custom configuration file
./DockerOps pull --config custom-config.json nginx:latest

# Specify username and password (for private registries)
./DockerOps pull --username myuser --password mypass private/image:tag

# Add prefix
./DockerOps pull --prefix myregistry.com/ nginx:latest
```

### Other Commands

```bash
# Search for images
./DockerOps search nginx
./DockerOps search --arch amd64 tensorflow

# Check version
./DockerOps version

# Show help
./DockerOps help

# Show specific command help
./DockerOps pull --help
./DockerOps search --help
```

## 🔍 Image Search

DockerOps provides powerful image search capabilities using advanced API integration:

### Search Features

- **Multi-Registry Search** - Search across multiple registries simultaneously
- **Architecture Filtering** - Filter results by specific architectures (amd64, arm64, etc.)
- **Detailed Information** - Get comprehensive image details including size, platform, creation time
- **Mirror Discovery** - Automatically discover available mirrors for images

### Search Examples

```bash
# Basic search
./DockerOps search nginx

# Search with architecture filter
./DockerOps search --arch amd64 tensorflow

# Search for specific versions
./DockerOps search python:3.9

# Search for AI/ML images
./DockerOps search pytorch
./DockerOps search --arch arm64 tensorflow
```

### Search Output

The search command provides detailed information for each found image:

```
找到 3 个匹配的镜像:
================================================================================

[1] docker.io/nginx:latest
    镜像源: swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/nginx:latest
    平台: linux/amd64
    大小: 187MB
    创建时间: 2024-01-15T10:30:00Z

[2] docker.io/nginx:alpine
    镜像源: registry.cn-hangzhou.aliyuncs.com/nginx:alpine
    平台: linux/amd64
    大小: 23MB
    创建时间: 2024-01-10T08:15:00Z
```

## ⚙️ Configuration

DockerOps uses a `config.json` file to manage image registry configurations. The default configuration includes:

- Alibaba Cloud Container Registry
- Tencent Cloud Container Registry
- Huawei Cloud Container Registry
- Other public registries

### Configuration File Format

```json
{
  "registries": [
    {
      "name": "Alibaba Cloud",
      "url": "registry.cn-hangzhou.aliyuncs.com",
      "priority": 1,
      "auth_required": false,
      "timeout": 15,
      "description": "Alibaba Cloud Container Registry"
    }
  ]
}
```

### Configuration Fields

- `name`: Registry name
- `url`: Registry URL
- `priority`: Priority (lower numbers have higher priority)
- `auth_required`: Whether authentication is required
- `timeout`: Timeout in seconds
- `description`: Registry description

## 🔌 API Reference

DockerOps also provides public API interfaces. For detailed information, please refer to the [API Documentation](api/refer.md).

Main API endpoints:

- `GET /api/v1/latest` - Get latest sync
- `GET /api/v1/image?search=<image-name>` - Search images
- `GET /api/v1/health` - Health check

## 🏗️ Project Structure

```
DockerOps/
├── cmd/                    # Command line interface
│   └── root.go            # Root command and subcommand definitions
├── internal/              # Internal packages
│   ├── config/           # Configuration management
│   └── puller/           # Image pulling logic
├── api/                   # API documentation
│   └── refer.md          # API reference documentation
├── build/                 # Build output directory
├── .github/              # GitHub Actions workflows
├── config.json           # Default configuration file
├── build.bat             # Windows build script
├── build.sh              # Linux/macOS build script
├── main.go               # Program entry point
├── go.mod                # Go module file
├── go.sum                # Go dependency verification file
├── LICENSE               # License file
└── README.md             # Project documentation
```

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) - Powerful CLI library
- [ProgressBar](https://github.com/schollz/progressbar) - Progress bar display
- Various cloud service providers for their registry services

## 📞 Contact

If you have any questions or suggestions, please contact us through:

- Submit an [Issue](https://github.com/caapap/DockerOps/issues)
- Send email to: caapap@qq.com

---

⭐ If this project helps you, please give it a star! 
