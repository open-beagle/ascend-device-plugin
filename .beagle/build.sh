#!/usr/bin/env bash
set -ex

# 初始化并全量更新所有的 Git 子模块 (例如 mind-cluster)
git config --global url."https://gitcode.com/".insteadOf "git@gitcode.com:" || true
git submodule update --init --recursive

# 配置 Aliyun APT 加速源
sed -i -e 's/deb.debian.org/mirrors.aliyun.com/g' -e 's/security.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list /etc/apt/sources.list.d/debian.sources 2>/dev/null || true
sed -i -e 's/archive.ubuntu.com/mirrors.aliyun.com/g' -e 's/security.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || true

# 安装 ARM64 交叉编译链（因为 Ascend 的 dcmi 库依赖 CGO 且只支持目标架构编译）
apt-get update -y && apt-get install -y gcc-aarch64-linux-gnu

export CGO_ENABLED=1
export CC=aarch64-linux-gnu-gcc
export GOOS=linux
export GOARCH=arm64
export GOPROXY="https://goproxy.cn,direct"

# 严格遵循上游官方 Dockerfile 的依赖获取方式（严禁 go mod tidy 导致依赖风暴）
go mod download github.com/Project-HAMi/HAMi
go get github.com/Project-HAMi/ascend-device-plugin/internal/server
go get huawei.com/npu-exporter
go get huawei.com/npu-exporter/utils/logger@v0.0.0-00010101000000-000000000000

# 从流水线环境变量捕获传入的版本号
export VERSION=${BUILD_VERSION:-v1.2.0-beagle}

# 遵循官方构建标准，一键调用 Makefile
make all
