@echo off
chcp 65001 >nul

REM DockerOps Windows 构建脚本

set VERSION=v0.1
set APP_NAME=DockerOps
set BUILD_DIR=build

echo 清理构建目录...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
mkdir %BUILD_DIR%

echo.
echo 开始构建 DockerOps %VERSION%
echo ==========================

REM 设置构建标志
set LDFLAGS=-s -w -X dockerops/cmd.VERSION=%VERSION%

echo 构建 Windows amd64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o %BUILD_DIR%\%APP_NAME%-%VERSION%-windows-amd64.exe main.go
if errorlevel 1 (
    echo 构建 Windows amd64 失败
    exit /b 1
)
echo ✅ 构建完成: %BUILD_DIR%\%APP_NAME%-%VERSION%-windows-amd64.exe

echo 构建 Linux amd64...
set GOOS=linux
set GOARCH=amd64
go build -ldflags="%LDFLAGS%" -o %BUILD_DIR%\%APP_NAME%-%VERSION%-linux-amd64 main.go
if errorlevel 1 (
    echo 构建 Linux amd64 失败
    exit /b 1
)
echo ✅ 构建完成: %BUILD_DIR%\%APP_NAME%-%VERSION%-linux-amd64

echo 构建 Linux arm64...
set GOOS=linux
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o %BUILD_DIR%\%APP_NAME%-%VERSION%-linux-arm64 main.go
if errorlevel 1 (
    echo 构建 Linux arm64 失败
    exit /b 1
)
echo ✅ 构建完成: %BUILD_DIR%\%APP_NAME%-%VERSION%-linux-arm64

echo 构建 macOS arm64...
set GOOS=darwin
set GOARCH=arm64
go build -ldflags="%LDFLAGS%" -o %BUILD_DIR%\%APP_NAME%-%VERSION%-darwin-arm64 main.go
if errorlevel 1 (
    echo 构建 macOS arm64 失败
    exit /b 1
)
echo ✅ 构建完成: %BUILD_DIR%\%APP_NAME%-%VERSION%-darwin-arm64

echo.
echo 🎉 所有平台构建完成！
echo 构建文件位于: %BUILD_DIR%\
echo.
echo 文件列表:
dir %BUILD_DIR%

echo.
echo 🚀 构建完成！可以在 %BUILD_DIR% 目录中找到所有平台的可执行文件。

pause 