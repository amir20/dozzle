---
title: Podman
sourceHash: 6baf154c7545
---

# Podman

Dozzle 通过 Podman 的 Docker 兼容套接字接口来支持 Podman。有两个已知的与 Docker 的差异会影响配置：在 rootless/Quadlet 部署中内存统计经常缺失（cgroup 委派问题），以及 Podman 不会生成 engine-id。本指南涵盖独立模式（本地监控）和代理模式（通过中心 Dozzle 服务器进行远程监控）。

## 部署方式

| 模式     | 适用场景       | 配置复杂度 |
| -------- | -------------- | ---------- |
| **独立** | 单主机日志查看 | 简单       |
| **代理** | 多主机集中监控 | 中等       |

### 启动方式

Podman 提供了几种启动方式：

| 方式               | 自动启动 | 内存统计 | 健康检查 | 最适合 |
| ------------------ | -------- | -------- | -------- | ------ |
| CLI                | 手动     | ✓        | ✓        | 开发   |
| `podman-compose`   | ✗        | ✓        | ✗        | 测试   |
| Quadlet（systemd） | ✓        | ✗\*      | ✓        | 生产   |

\*在 rootless 模式下，除非启用了 cgroup v2 内存委派，否则内存统计通常不可用。参见本页底部的 FAQ。

---

# <Icon icon="mdi:monitor-dashboard" inline /> 独立模式

以独立服务的方式运行 Dozzle，监控本地的 Podman 容器。

## <Icon icon="mdi:shield-account-outline" inline /> Rootful 配置

适用于系统级的 Podman 守护进程：

```bash
# 启用并启动 Podman 套接字
sudo systemctl enable podman.socket
sudo systemctl start podman.socket

# Dozzle 可以通过 Docker 套接字连接
podman run -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

## <Icon icon="mdi:account-outline" inline /> Rootless 配置

Rootless Podman 把容器隔离在一个用户命名空间中：

```bash
# 启动用户级套接字（随用户会话自动运行）
systemctl --user enable podman.socket
systemctl --user start podman.socket

# 对于名为 'appuser' 的用户，Dozzle 可以这样连接：
podman run -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

**重要**：绑定到某个用户 rootless 套接字的 Dozzle 只能看到该用户的容器。其他用户的 rootless 容器位于各自独立的命名空间中，不会出现在这里。

## <Icon icon="mdi:rocket-launch-outline" inline /> Quadlet 部署

Quadlet 提供了 systemd 原生的容器管理方式。在 `~/.config/containers/systemd/dozzle.container` 创建一个 `.container` 文件：

```ini
[Unit]
Description=Dozzle Log Viewer
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

启用并启动：

```bash
systemctl --user daemon-reload
systemctl --user enable --now dozzle.service
```

在多用户系统上，把同一个文件放进每个用户的 `~/.config/containers/systemd/`，并为每个用户选一个不同的主机端口（例如 `PublishPort=3001:8080`）。每个实例只能看到对应用户的 rootless 容器。

> [!NOTE] Quadlet 会为健康检查生成一个 systemd 定时器。`podman-compose` 不会，所以在那里健康检查不会按计划运行；需要时可以用 `podman healthcheck run NAME` 手动触发。

---

# <Icon icon="mdi:lan-connect" inline /> 代理模式

在远程 Podman 主机上以代理方式运行 Dozzle，通过一台主 Dozzle 服务器实现集中监控。代理通过 gRPC 与主服务器通信。

## <Icon icon="mdi:cog-outline" inline /> 代理配置

### 前置条件

- 在代理主机上开放 7007 端口
- 主服务器与代理之间网络可达

### 启动 Dozzle 代理

在远程 Podman 主机上以代理模式运行 Dozzle：

```bash
# Rootful 代理
podman run -d \
  --name dozzle-agent \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

```bash
# Rootless 代理（用户为 'appuser'）
sudo -u appuser podman run -d \
  --name dozzle-agent \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

### Quadlet 代理部署

为代理创建一个 `.container` 文件：

```ini
# dozzle-agent.container
[Unit]
Description=Dozzle Agent
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=7007:7007
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro
Exec=agent

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] Dozzle 镜像的 entrypoint 是 `/dozzle`，所以 `agent` 要写在 `Exec=`（命令）里，而不是 `Entrypoint=`。

启用并启动：

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# <Icon icon="mdi:server-network" inline /> 带远程代理的主服务器

配置主 Dozzle 服务器，让它连接到远程 Podman 主机上的代理。

## <Icon icon="mdi:cog" inline /> 服务器配置

带上代理端点运行主 Dozzle 服务器：

```bash
podman run -d \
  --name dozzle \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest \
  --remote-agent "host1.example.com:7007" \
  --remote-agent "host2.example.com:7007"
```

或者使用环境变量：

```bash
podman run -d \
  --name dozzle \
  -e DOZZLE_REMOTE_AGENT="host1.example.com:7007,host2.example.com:7007" \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

### 使用 Quadlet 部署带代理的主服务器

```ini
# dozzle-server.container
[Unit]
Description=Dozzle Server with Remote Agents
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Environment=DOZZLE_REMOTE_AGENT=host1.example.com:7007,host2.example.com:7007

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] `WantedBy=multi-user.target` 只适用于系统单元。对于 `systemctl --user` 单元，请使用 `default.target`。

---

# <Icon icon="mdi:tune" inline /> 其他配置

## <Icon icon="mdi:identifier" inline /> engine-id 配置

Podman 不会像 Docker 那样创建 engine-id。创建一个可以避免“host not found”错误：

### 使用 uuidgen

```bash
# 如有需要，先创建目录
sudo mkdir -p /var/lib/docker

# 生成 UUID
sudo sh -c 'uuidgen > /var/lib/docker/engine-id'

# 验证
cat /var/lib/docker/engine-id
```

### 使用 Ansible

```yaml
- name: Create /var/lib/docker
  ansible.builtin.file:
    path: /var/lib/docker
    state: directory
    mode: "755"

- name: Create engine-id and derive UUID from hostname
  ansible.builtin.lineinfile:
    path: /var/lib/docker/engine-id
    line: "{{ hostname | to_uuid }}"
    create: true
    mode: "0644"
    insertafter: "EOF"
```

> [!WARNING] 在带上 engine-id 重新创建之前，请先清理已有的 Dozzle 部署（停止容器、删除卷）。

## <Icon icon="mdi:help-circle-outline" inline /> FAQ

### rootless 模式下缺少内存统计

在 rootless 部署中通常看不到内存统计，因为 `memory` cgroup 控制器默认没有被委派给用户 slice。检查已委派的控制器：

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

如果输出中没有 `memory`，可以通过一个 drop-in 文件启用委派：

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf <<'EOF'
[Service]
Delegate=cpu cpuset io memory pids
EOF
sudo systemctl daemon-reload
```

然后注销再登录（或重启），让用户 slice 应用新的委派配置。详情参见 [Podman rootless 教程](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md)。

### 健康检查被报告为 unhealthy

**podman-compose 的问题**：即使手动运行能通过，健康检查仍被报告为 unhealthy。这是 Podman 的行为：没有 systemd 定时器时，健康检查不会被自动执行（Quadlet 会自动生成一个）。

使用 `podman-compose` 时的变通办法：

```bash
# 手动运行健康检查
podman healthcheck run <container_id>
```

**Quadlet**：`HealthCmd=` 接受的是普通命令行，而不是 Docker 的 `CMD [...]` JSON 形式：

```ini
HealthCmd=/dozzle healthcheck
```

较老的 `podman-compose`（< 1.5.0）会用 `sh` 运行所有健康检查，而 Dozzle 镜像里并没有 `sh`。请升级到较新的版本。

### 跨用户的容器可见性

Rootless Podman 只能访问同一个用户命名空间中的容器。如果 Dozzle 以某个用户身份运行，它看不到另一个用户 rootless 会话中的容器。

**解决办法**：要么让 Dozzle 以同一个用户运行，要么使用 rootful 模式。
