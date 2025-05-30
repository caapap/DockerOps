# DockerOps

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Version](https://img.shields.io/badge/Version-v2.0.0-orange.svg)](https://github.com/yourusername/DockerOps/releases)

**[中文文档 / Chinese Documentation](README_CN.md)**

DockerOps is an enhanced Docker image pulling tool designed to solve Docker image pulling difficulties in China. It supports multi-registry search, automatic failover, concurrent downloads, and more.

## ✨ Features

- 🚀 **Multi-Registry Support** - Supports multiple Chinese registries including Alibaba Cloud, Tencent Cloud, Huawei Cloud
- 🔄 **Automatic Failover** - Automatically switches to the next registry when one is unavailable
- ⚡ **Concurrent Downloads** - Supports multi-threaded concurrent downloads for improved speed
- 📊 **Progress Bar Display** - Real-time download progress visualization
- 🔧 **Configuration Management** - Manage image registries through JSON configuration files
- 🏷️ **Tag Transformation Rules** - Intelligent tag conversion and mapping
- 🌐 **Cross-Platform Support** - Supports Windows, Linux, macOS
- 🔍 **Image Search** - Search images across multiple registries
- 📦 **Image Management** - Support for image push, load, save operations

## 🛠️ Installation

### Build from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/DockerOps.git
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

Download the corresponding platform's pre-compiled binary from the [Releases](https://github.com/yourusername/DockerOps/releases) page.

## 📖 Usage

### Basic Usage

```bash
# Pull an image
./DockerOps pull nginx:latest

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
# Check version
./DockerOps version

# Show help
./DockerOps help

# Show specific command help
./DockerOps pull --help
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

- Submit an [Issue](https://github.com/yourusername/DockerOps/issues)
- Send email to: your-email@example.com

---

⭐ If this project helps you, please give it a star! 