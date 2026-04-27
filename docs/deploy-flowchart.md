# ascend-device-plugin 部署流程图

## 一、整体流程总览

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              整体部署流程                                        │
│                                                                                 │
│  ① beagle-smi-npu 编译  →  ② 二进制拷贝到 ascend-device-plugin                  │
│          ↓                            ↓                                         │
│  ③ ascend-device-plugin 编译 + 打包镜像  →  ④ kubectl apply 部署到 K8s           │
│                                                    ↓                            │
│                                       ⑤ DaemonSet 在节点上运行                   │
│                                            ↓                                    │
│                                  ⑥ initContainer 安装 beagle-smi-npu 到宿主机    │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## 二、beagle-smi-npu 编译 & 二进制传递流程

```
┌──────────────────────────────────────┐
│     beagle-smi-npu 项目 (子模块)      │
│                                      │
│  源码目录:                            │
│    beagle-smi-npu/                   │
│    ├── cmd/app/          # 入口      │
│    ├── internal/         # 核心逻辑   │
│    └── scripts/build.sh  # 编译脚本   │
│                                      │
│  执行编译:                            │
│    cd beagle-smi-npu                 │
│    bash scripts/build.sh             │
│                                      │
│  编译产物:                            │
│    beagle-smi-npu/dist/              │
│    ├── beagle-smi-npu-linux-arm64    │
│    └── beagle-smi-npu-linux-amd64    │
└──────────────┬───────────────────────┘
               │
               │  手动拷贝 (cp 命令)
               │  cp dist/beagle-smi-npu-linux-arm64 \
               │     ../ascend-device-plugin/dist/beagle-smi-npu
               ▼
┌──────────────────────────────────────┐
│   ascend-device-plugin 项目          │
│                                      │
│  二进制存放位置:                       │
│    ascend-device-plugin/             │
│    └── dist/                         │
│        └── beagle-smi-npu  ◄── 这里  │
│                                      │
└──────────────────────────────────────┘
```

## 三、ascend-device-plugin 构建 & 镜像打包流程

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CI 流水线 (.beagle/build.sh)                      │
│                                                                     │
│  1. go build → 编译 ascend-device-plugin 二进制                      │
│  2. cp ./dist/beagle-smi-npu ./beagle-smi-npu                      │
│     (将预置的 beagle-smi-npu 拷贝到构建根目录)                        │
│                                                                     │
│  构建根目录产物:                                                      │
│    ./ascend-device-plugin    ← Go 编译产物                           │
│    ./beagle-smi-npu          ← 从 dist/ 拷贝来的                     │
└────────────────────────┬────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Docker 镜像 (.beagle/dockerfile)                  │
│                                                                     │
│  FROM ubuntu:22.04                                                  │
│                                                                     │
│  COPY ascend-device-plugin /usr/local/bin/ascend-device-plugin      │
│  COPY beagle-smi-npu       /usr/local/bin/beagle-smi-npu           │
│                                                                     │
│  镜像内文件:                                                         │
│    /usr/local/bin/ascend-device-plugin  ← 主程序                     │
│    /usr/local/bin/beagle-smi-npu       ← SMI 替换工具                │
│                                                                     │
│  镜像地址:                                                           │
│    registry.cn-qingdao.aliyuncs.com/wod/ascend-device-plugin        │
│    tag: v1.2.0-smi-arm64                                            │
└─────────────────────────────────────────────────────────────────────┘
```

## 四、K8s 部署流程 & YAML 文件位置

```
┌─────────────────────────────────────────────────────────────────────┐
│                     部署 YAML 文件                                   │
│                                                                     │
│  项目中位置:                                                         │
│    ascend-device-plugin/ascend-device-plugin.yaml                   │
│                                                                     │
│  使用方式 (在能访问 K8s 集群的机器上执行):                              │
│    kubectl apply -f ascend-device-plugin.yaml                       │
│                                                                     │
│  说明:                                                               │
│    该文件不需要放到服务器固定位置，只需在有 kubectl 权限的               │
│    机器上执行 apply 即可。文件通过 kubectl 提交给 K8s API Server，     │
│    K8s 会自动将 DaemonSet 调度到所有 ascend="on" 标签的节点。          │
│                                                                     │
│  建议存放位置 (运维管理):                                              │
│    /opt/k8s/manifests/ascend-device-plugin.yaml                     │
│    或项目仓库中直接使用，无需额外拷贝                                   │
└─────────────────────────────────────────────────────────────────────┘
```

## 五、DaemonSet 节点运行流程

```
┌─────────────────────────────────────────────────────────────────────┐
│              K8s 节点 (带 ascend="on" 标签)                          │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────┐      │
│  │  Pod: hami-ascend-device-plugin                           │      │
│  │                                                           │      │
│  │  Step 1: initContainer (install-beagle-smi-npu)           │      │
│  │  ┌─────────────────────────────────────────────────────┐  │      │
│  │  │ 1. mkdir -p /opt/beagle-smi (宿主机)                │  │      │
│  │  │ 2. 备份原始 npu-smi:                                │  │      │
│  │  │    cp /usr/local/sbin/npu-smi                       │  │      │
│  │  │       → /opt/beagle-smi/npu-smi.real (仅首次)       │  │      │
│  │  │ 3. 安装 beagle-smi-npu:                             │  │      │
│  │  │    镜像内 /usr/local/bin/beagle-smi-npu             │  │      │
│  │  │       → /opt/beagle-smi/npu-smi (宿主机)            │  │      │
│  │  │       → /usr/local/sbin/npu-smi (宿主机，替换原始)   │  │      │
│  │  └─────────────────────────────────────────────────────┘  │      │
│  │                                                           │      │
│  │  Step 2: 主容器 (device-plugin)                           │      │
│  │  ┌─────────────────────────────────────────────────────┐  │      │
│  │  │ 1. 注册 Device Plugin 到 kubelet                    │  │      │
│  │  │ 2. Allocate 时自动注入到用户 Pod:                    │  │      │
│  │  │    - env: huawei.com/Ascend910B2-memory             │  │      │
│  │  │    - env: REAL_NPU_SMI_PATH=/opt/beagle-smi/...     │  │      │
│  │  │    - mount: /opt/beagle-smi (hostPath)              │  │      │
│  │  └─────────────────────────────────────────────────────┘  │      │
│  └───────────────────────────────────────────────────────────┘      │
│                                                                     │
│  宿主机文件结构:                                                     │
│    /opt/beagle-smi/                                                 │
│    ├── npu-smi       ← beagle-smi-npu (替换版)                      │
│    └── npu-smi.real  ← 原始 npu-smi (备份)                          │
│    /usr/local/sbin/                                                 │
│    └── npu-smi       ← beagle-smi-npu (替换版，全局可用)             │
└─────────────────────────────────────────────────────────────────────┘
```

## 六、完整路径汇总

| 阶段 | 文件 | 路径 |
|------|------|------|
| beagle-smi-npu 编译产物 | arm64 二进制 | `beagle-smi-npu/dist/beagle-smi-npu-linux-arm64` |
| 拷贝到 ascend-device-plugin | 预置二进制 | `ascend-device-plugin/dist/beagle-smi-npu` |
| CI 构建时拷贝到根目录 | 构建临时文件 | `./beagle-smi-npu` (构建根目录) |
| Docker 镜像内 | 主程序 | `/usr/local/bin/ascend-device-plugin` |
| Docker 镜像内 | SMI 工具 | `/usr/local/bin/beagle-smi-npu` |
| K8s 部署 YAML | 部署清单 | `kubectl apply -f ascend-device-plugin.yaml` |
| 宿主机 | beagle-smi-npu | `/opt/beagle-smi/npu-smi` |
| 宿主机 | 原始 npu-smi 备份 | `/opt/beagle-smi/npu-smi.real` |
| 宿主机 | 全局 npu-smi (替换) | `/usr/local/sbin/npu-smi` |
