# Mini Envoy

面向自研二进制协议的高性能反向代理与可视化控制台。生产数据面使用 Linux Epoll 边沿触发 Reactor；macOS 使用 Kqueue 做功能验证。

## 1. 如何启动

```bash
docker compose up --build -d
```

控制台：http://127.0.0.1:31881/  
数据面：`127.0.0.1:31880`（MENV v1）

无需在宿主机安装 Go / Node。时区为 `Asia/Shanghai`。

## 2. 使用说明

打开控制台可查看并发连接、PPS、吞吐示波器、环形缓冲占用、连接池健康仪和 Client→Gateway→Route→Upstream 拓扑。在上游表中摘除或恢复节点；危险操作使用自定义确认框。

协议探测：

```bash
go run ./cmd/menvctl ping --target 127.0.0.1:31880
go run ./cmd/menvctl echo --target 127.0.0.1:31880 --route 1 --payload hello
```

`upstream-c` 默认每 8 个请求注入一次 ERROR，用于观察故障可见性。

## 3. 服务列表及API说明

| 入口 | 说明 |
| --- | --- |
| http://127.0.0.1:31881/ | Vue 控制台（Nginx 同源反代 `/api` `/healthz`） |
| http://127.0.0.1:31881/healthz | 进程健康 |
| http://127.0.0.1:31881/api/v1/overview | 指标快照 |
| http://127.0.0.1:31881/api/v1/routes | 路由 |
| http://127.0.0.1:31881/api/v1/upstreams | 上游与健康 |
| POST /api/v1/upstreams/{id}/eject\|restore | 人工摘除/恢复 |
| GET /api/v1/topology | 拓扑 |
| GET /api/v1/events | SSE |
| `127.0.0.1:31880` | MENV 数据面 |

字段级契约见 `docs/API.md`、`docs/Protocol.md`。

## 4. 测试账号

单运维者、无登录。演示管理令牌由 Nginx 注入（`demo-minienvoy-token`），浏览器无需填写。本地 `configs/local.yaml` 绑定 `127.0.0.1` 时可不设 Token。

## 5. 题目内容

实现零额外复制（应用内）网关：Epoll/Kqueue Reactor、环形缓冲、MENV v1、RR/SWRR、健康与一键摘除、Vue 实时控制台。完整范围见 `docs/Requirements.md`。

## 6. 项目结构

```text
cmd/                 minienvoy / menvctl / mockupstream
internal/            platform reactor buffer protocol proxy routing upstream probe metrics admin config
web/                 Vue 3 控制台
configs/             demo.yaml local.yaml
deploy/              Dockerfile.* nginx.conf
docs/                需求、路线图、API、协议、架构
```

## 7. API 模拟与切换指南

本项目没有第三方计量 API。`cmd/mockupstream` 是讲 MENV v1 的真实协议进程，用来组成可演示的三节点集群，不是伪造管理接口。

- **演示模式（默认）**：`docker compose up` 将 `upstream-a/b/c` 作为网关上游。`upstream-c` 的 `MENV_MODE=flaky` 会注入可见 ERROR。
- **接入真实上游**：编辑 `configs/demo.yaml` 的 `upstreams.host/port`，指向你自己的 MENV 服务，然后 `docker compose up -d --force-recreate gateway`。数据面只消费解析后的 IPv4，主机名在控制面解析。
- **关闭故障注入**：去掉 `upstream-c` 的 `MENV_MODE=flaky`，或把它从路由成员中删除。
- **本地功能调试**：`go run ./cmd/mockupstream` + `go run ./cmd/minienvoy -config configs/local.yaml`。

管理 API 始终反映网关真实状态，前端禁止随机健康数据。
