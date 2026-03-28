#!/usr/bin/env bash
set -ex

# 初始化并全量更新所有的 Git 子模块 (例如 mind-cluster)
git config --global url."https://gitcode.com/".insteadOf "git@gitcode.com:" || true
git submodule update --init --recursive

export CGO_ENABLED=0
export GOOS=linux
export GOARCH=arm64
export GOPROXY="https://goproxy.cn,direct"

# 下载并整理依赖
go mod tidy

# 编译出适用于 CANN 的 ARM64 架构二进制程序
go build -ldflags "-s -w -X github.com/Project-HAMi/ascend-device-plugin/version.version=${BUILD_VERSION:-unknown}" -o ./ascend-device-plugin ./cmd/main.go
