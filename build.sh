#!/bin/bash

# DockerOps 跨平台构建脚本
# 支持 Windows, Linux, macOS

VERSION="v2.0.0"
APP_NAME="DockerOps"
BUILD_DIR="build"

# 清理构建目录
echo "清理构建目录..."
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR

# 构建信息
echo "开始构建 DockerOps $VERSION"
echo "=========================="

# 设置构建标志
LDFLAGS="-s -w -X dockerops/cmd.VERSION=$VERSION"

# 构建目标平台
platforms=(
    "windows/amd64"
    "linux/amd64"
    "linux/arm64"
    "darwin/arm64"
)

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    
    output_name=$APP_NAME'-'$VERSION'-'$GOOS'-'$GOARCH
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "构建 $GOOS/$GOARCH..."
    
    env GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="$LDFLAGS" -o $BUILD_DIR/$output_name main.go
    
    if [ $? -ne 0 ]; then
        echo "构建 $GOOS/$GOARCH 失败"
        exit 1
    fi
    
    echo "✅ 构建完成: $BUILD_DIR/$output_name"
done

echo ""
echo "🎉 所有平台构建完成！"
echo "构建文件位于: $BUILD_DIR/"
echo ""
echo "文件列表:"
ls -la $BUILD_DIR/

# 创建发布包
echo ""
echo "创建发布包..."
cd $BUILD_DIR

for file in *; do
    if [[ $file == *.exe ]]; then
        # Windows 版本
        zip "${file%.*}.zip" "$file" ../config.json ../README.md
    else
        # Unix 版本
        tar -czf "${file}.tar.gz" "$file" ../config.json ../README.md
    fi
    echo "📦 创建发布包: ${file%.*}"
done

cd ..
echo ""
echo "🚀 发布包创建完成！" 