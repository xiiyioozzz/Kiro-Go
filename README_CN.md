# Kiro-Go Enhanced

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Kiro-Go Enhanced 可以把你有权管理的 Kiro 账号转换成 OpenAI / Anthropic 兼容 API 服务，并提供一个 Web 后台来管理账号导入、额度同步、请求日志、模型刷新和账号池状态。

[English](README.md) | 中文

本项目基于 [zsecducna/Kiro-Go](https://github.com/zsecducna/Kiro-Go) 增强开发，保留原项目的核心目标：让 Kiro 账号可以通过通用 API 协议被客户端调用。本增强版更偏向“账号池运营”：更多导入渠道、更清楚的账号健康状态、更适合大量账号的后台界面，以及更完整的日志与统计。

请只使用你自己拥有或被授权管理的账号和凭证。你需要自行遵守 Kiro、AWS、Amazon 以及各身份提供商的服务条款。

## 项目能做什么

- 提供 Claude、OpenAI Chat Completions、OpenAI Responses 兼容接口。
- 将请求分配到 Kiro 账号池，并处理刷新、失败切换、统计和账号状态。
- 在 `/admin` 提供后台面板，用于添加账号、测试账号、刷新额度/模型和查看请求历史。
- 支持多种账号来源：Kiro OAuth 流程、本地缓存、凭证 JSON、浏览器 Cookie、Kiro API Key 等。
- 默认把运行数据保存到 `data/config.json`，Docker、本地源码运行和托管部署都可以使用同一套账号池数据。

## 相比 zsecducna/Kiro-Go 的修改和提升

| 方向 | 上游项目重点 | 本增强版 |
| --- | --- | --- |
| 账号导入 | 基础 Kiro 登录和导入流程 | 增加 Microsoft / Entra ID SSO、Google/GitHub 托管登录、Kiro API Key 导入、更宽松的凭证 JSON 兼容、本地缓存导入工作流 |
| 账号健康 | 基础启用/禁用与刷新状态 | 统计可调用账号、封禁/暂停/额度耗尽/认证失败/Profile 异常/同步失败/Token 过期等状态，并展示最后失败原因 |
| 账号运营 | 单账号管理和基础批量动作 | 增加批量测试、批量刷新模型、账号去重、添加时间、Region 编辑/检测、权重、代理和超额调用控制 |
| API 兼容 | Claude 和 OpenAI Chat Completions | 增加 OpenAI Responses API、`/v1/stats`、模型聚合、多 API Key 管理、每个 Key 的用量统计和限制、Thinking 模式格式配置 |
| 请求日志 | 请求日志查看 | 区分客户端来源和 Kiro 上游端点，记录账号快照，显示更清楚的错误分类，表格更适合快速扫描 |
| 后台界面 | 可用的后台管理面板 | 重新设计添加账号入口和图标，顶部导航/看板/账号列表/批量工具栏/搜索筛选可置顶，账号统计悬浮层更清楚，中英文文案更完整 |
| Region/Profile | 基础 Region 刷新逻辑 | 增加账号级 Region 显示、手动切换、自动检测、Profile ARN 保留，以及可配置的 Profile Region 探测列表 |
| 安全与发布 | 运行配置持久化 | 忽略本地 `data/` 和 `auth-output/`，校验外部 IdP Token Endpoint，拦截常见重复导入和错误导入 |

## 核心功能

- API 接口：
  - Claude Messages：`/v1/messages`
  - Claude Count Tokens：`/v1/messages/count_tokens`
  - OpenAI Chat Completions：`/v1/chat/completions`
  - OpenAI Responses：`/v1/responses`
  - 模型列表：`/v1/models`
  - 运行统计：`/v1/stats`
- 账号导入方式：
  - AWS Builder ID
  - IAM Identity Center / Enterprise SSO
  - Microsoft 365 / Entra ID 托管 SSO
  - Google/GitHub 托管 Kiro 登录
  - Kiro API Key（`ksk_...`）
  - SSO Token（`x-amz-sso_authn`）
  - Kiro IDE 本地缓存
  - Kiro Account Manager / 凭证 JSON
  - Kiro 网页 Cookie RefreshToken
- 账号池管理：
  - 轮询调度和失败切换
  - 单账号启用、禁用、删除、刷新、刷新模型、测试
  - 批量启用、禁用、刷新、刷新模型、测试、删除
  - 全局代理和单账号代理
  - 账号权重，用于提高部分账号调度优先级
  - 额度、用量、Credits、超额调用和订阅信息同步
- 后台体验：
  - 顶部导航、统计看板、账号列表标题、批量工具栏、搜索和筛选置顶
  - 账号数量显示为 `可调用/总账号`，同时展示总 Credits
  - 悬停统计可查看账号可用性、Token 状态、额度状态和同步状态
  - 请求日志展示来源端点、上游端点、模型、账号、Tokens、耗时、Credits 和错误分类
  - 中英文界面

## 快速开始

### Docker Compose

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
mkdir -p data
docker compose up -d --build
```

打开 `http://127.0.0.1:8080/admin`。

Compose 文件会把 Microsoft / Kiro 托管 SSO 回调端口发布为 `127.0.0.1:3128:3128`，这样 Docker 宿主机上的浏览器可以完成登录回调，同时不会把回调端口暴露到外部网络。

### Docker Run

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
mkdir -p data
docker build -t kiro-go:codex-enhanced .

docker run -d \
  --name kiro-go \
  -p 8080:8080 \
  -p 127.0.0.1:3128:3128 \
  -e CONFIG_PATH=/app/data/config.json \
  -e ADMIN_PASSWORD=change_this_password \
  -e KIRO_SSO_CALLBACK_BIND=0.0.0.0 \
  -v "$(pwd)/data:/app/data" \
  --restart unless-stopped \
  kiro-go:codex-enhanced
```

### 源码编译

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
go build -o kiro-go .
CONFIG_PATH=data/config.json ADMIN_PASSWORD=change_this_password ./kiro-go
```

## 基本使用

1. 打开 `http://127.0.0.1:8080/admin`。
2. 使用后台密码登录。如果没有设置，初始默认密码是 `changeme`。
3. 在添加账号弹窗中导入一个或多个 Kiro 账号。
4. 刷新账号额度和模型列表。
5. 使用兼容接口发起请求。

Claude 兼容请求：

```bash
curl http://127.0.0.1:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4.5","max_tokens":1024,"messages":[{"role":"user","content":"你好！"}]}'
```

OpenAI Chat Completions 兼容请求：

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer any" \
  -d '{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"你好！"}]}'
```

OpenAI Responses 兼容请求：

```bash
curl http://127.0.0.1:8080/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer any" \
  -d '{"model":"claude-sonnet-4.5","input":"你好！"}'
```

## 配置

运行配置默认保存在 `data/config.json`。Docker 运行时建议挂载或备份 `/app/data`。

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| `CONFIG_PATH` | 配置文件路径 | `data/config.json` |
| `ADMIN_PASSWORD` | 后台管理密码覆盖值 | 配置文件中的值 |
| `LOG_LEVEL` | 日志等级，例如 `debug`、`info`、`warn`、`error` | `info` |
| `KIRO_SSO_CALLBACK_BIND` | 托管 SSO 临时回调监听地址 | 仅本机回环 |
| `KIRO_PROFILE_REGIONS` | Profile Region 探测列表，多个 Region 用英文逗号分隔 | 内置 Region 列表 |

## API 鉴权

后台可以创建多个代理 API Key。开启 API Key 验证后，客户端请求需要携带以下任意一种请求头：

- `Authorization: Bearer <key>`
- `X-Api-Key: <key>`

每个 Key 都可以独立设置启用状态、Token/Credit 限制、用量统计和最后使用时间。

## Thinking 模式

在模型名后追加配置的后缀即可启用，默认后缀是 `-thinking`，例如 `claude-sonnet-4.5-thinking`。

Claude 兼容请求如果带有顶层 `thinking` 配置，例如 `{"type":"enabled","budget_tokens":2048}` 或 `{"type":"adaptive"}`，也会自动启用 Thinking 模式。输出格式可以在后台设置中调整。

## 出站代理

可以在设置中配置全局出站代理，也可以在账号详情里给单个账号配置代理。支持 SOCKS5 和 HTTP，保存后无需重启。

## 部署注意事项

- 对外暴露服务前必须设置强后台密码 `ADMIN_PASSWORD`。
- 如果 `/v1/*` 接口能被其他机器访问，建议开启代理 API Key 验证。
- 不要提交 `data/`、`auth-output/`、导出的凭证 JSON、浏览器缓存转储文件。
- 除非非常明确自己在做什么，否则不要把 SSO 回调端口 `3128` 绑定到公网地址。
- Refresh Token、Access Token、Cookie、Kiro API Key、导出的账号 JSON 都应当视为敏感凭证。

## 同步上游

本 fork 保留 `zsecducna/Kiro-Go` 作为 `upstream`。常见维护流程：

```bash
git fetch upstream
git checkout codex-enhanced
git merge upstream/feat/azure-tenant-sso
```

合并冲突需要谨慎处理，完成后建议运行测试并重新构建 Docker 镜像。

## 免责声明

本项目仅供学习和研究目的使用，与 Amazon、AWS、Kiro、Microsoft、Google、GitHub 没有任何从属或官方关系。用户需自行确保使用行为符合所有适用的服务条款、身份提供商政策和法律法规，使用风险自负。

## 许可证

[MIT](LICENSE)
