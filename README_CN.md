# Kiro-Go Enhanced

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Kiro-Go Enhanced 是一个本地或自托管的 Kiro 账号网关。它可以把你有权管理的 Kiro 账号转换成 Claude、OpenAI Chat Completions 和 OpenAI Responses 兼容 API，并提供一个 Web 后台来管理账号导入、额度同步、模型刷新、请求日志和账号池状态。

[English](README.md) | 中文

本仓库以 [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) 作为上游来源。当前分支定位为偏运营管理的版本，适合需要在一个后台里运行、观察和维护较大 Kiro 账号池的使用场景。

请只使用你自己拥有或被授权管理的账号和凭证。你需要自行遵守 Kiro、AWS、Amazon 以及各身份提供商的服务条款。

## 项目重点

- **兼容 API 网关**：通过常见的 Claude / OpenAI 风格 `/v1/*` 接口调用 Kiro-backed 模型能力。
- **账号池运营**：在一个后台里管理大量账号，支持启用/禁用、刷新、测试、删除、权重、代理、Region 和超额调用控制。
- **凭证导入中心**：支持托管 OAuth、SSO Token、本地 Kiro 缓存、凭证 JSON、浏览器 Cookie RefreshToken 和 Kiro API Key 等导入方式。
- **真实运行可视化**：显示可调用账号、总账号、Credits、额度状态、Token 状态、刷新失败、请求成功/失败、耗时和模型信息。
- **大量账号后台体验**：顶部导航、统计看板、账号列表控制区置顶，支持批量操作、搜索筛选、中英文界面和更清楚的账号状态标签。
- **部署安全习惯**：运行期敏感数据放在本地 `data/`，导出凭证不进入 git/docker，上游外部 IdP endpoint 会先校验再发送 refresh token。

## 主要能力

### API 兼容

- Claude Messages：`/v1/messages`
- Claude Count Tokens：`/v1/messages/count_tokens`
- OpenAI Chat Completions：`/v1/chat/completions`
- OpenAI Responses：`/v1/responses`
- 模型列表：`/v1/models`
- 运行统计：`/v1/stats`

### 账号导入方式

- AWS Builder ID
- IAM Identity Center / Enterprise SSO
- Microsoft 365 / Entra ID 托管 SSO
- Google/GitHub 托管 Kiro 登录
- Kiro API Key（`ksk_...`）
- 浏览器 `x-amz-sso_authn` SSO Token
- Kiro IDE 本地缓存
- Kiro Account Manager / 凭证 JSON
- Kiro 网页 Cookie RefreshToken

### 账号池管理

- 轮询调度和请求失败切换
- 自动 Token 刷新和刷新失败分类
- 单账号启用、禁用、删除、刷新、刷新模型、测试
- 批量启用、禁用、刷新、刷新模型、测试、删除
- 单账号路由权重
- 单账号代理和全局出站代理
- Region 显示、手动编辑和自动检测
- 订阅、额度、Credits、试用、超额调用和模型缓存信息
- 常见凭证格式的重复导入保护

### 后台管理

- 账号数量显示为 `可调用/总账号`
- 账号统计卡展示总 Credits
- 悬停统计展示可用性、禁用/封禁/暂停状态、Token 健康、额度状态、Profile 异常和同步失败
- 请求日志展示来源端点、Kiro 上游端点、模型、账号、Tokens、耗时、Credits 和错误分类
- 添加账号弹窗使用独立导入卡片和提供商图标
- 顶部导航、统计看板、账号列表标题、批量工具栏、搜索和筛选置顶
- 中英文界面

### 安全相关控制

- `data/`、`auth-output/`、本地日志和导出凭证不会进入 git 和 docker context。
- 外部 IdP issuer/token endpoint 必须是 HTTPS 且命中受支持 allow-list，才会发送 refresh token。
- 凭证导入接口有请求体大小限制。
- 托管 SSO 回调监听默认仅绑定本机回环地址，并且有时间限制。
- 可以要求所有 `/v1/*` 客户端请求携带代理 API Key。
- 代理 API Key 支持启用状态、Token/Credit 限制、用量统计和最后使用时间。

## 快速开始

### Docker Compose

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
mkdir -p data
docker compose up -d --build
```

打开 `http://127.0.0.1:8080/admin`。

Compose 文件会把托管 SSO 回调端口发布为 `127.0.0.1:3128:3128`，这样 Docker 宿主机上的浏览器可以完成登录回调，同时不会把回调端口暴露到外部网络。

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

## 首次使用

1. 打开 `http://127.0.0.1:8080/admin`。
2. 使用后台密码登录。如果没有设置，初始默认密码是 `changeme`。
3. 在添加账号弹窗中导入一个或多个 Kiro 账号。
4. 刷新账号额度和模型信息。
5. 在 IDE、CLI、脚本或兼容客户端中调用 `/v1/*` 接口。

## API 示例

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
- 不要提交 `data/`、`auth-output/`、导出的凭证 JSON、浏览器缓存转储文件或本地日志。
- 除非非常明确自己在做什么，否则不要把 SSO 回调端口 `3128` 绑定到公网地址。
- Refresh Token、Access Token、Cookie、Kiro API Key、导出的账号 JSON 都应当视为敏感凭证。

## 上游参考

本仓库以 [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) 作为上游参考。常见维护流程：

```bash
git fetch upstream
git checkout codex-enhanced
git merge upstream/main
```

合并冲突需要谨慎处理，完成后建议运行测试并重新构建 Docker 镜像。

## 免责声明

本项目仅供学习和研究目的使用，与 Amazon、AWS、Kiro、Microsoft、Google、GitHub 没有任何从属或官方关系。用户需自行确保使用行为符合所有适用的服务条款、身份提供商政策和法律法规，使用风险自负。

## 许可证

[MIT](LICENSE)
