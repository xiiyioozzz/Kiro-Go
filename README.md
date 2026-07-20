# Kiro-Go Enhanced

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Kiro-Go Enhanced turns your authorized Kiro accounts into a local or self-hosted OpenAI / Anthropic compatible API service, with a Web admin panel for account import, quota visibility, request logs, model discovery, and account-pool operations.

[English](README.md) | [中文](README_CN.md)

This fork is based on [zsecducna/Kiro-Go](https://github.com/zsecducna/Kiro-Go) and keeps the original goal intact: provide a practical compatibility layer for Kiro-backed model access. The enhanced branch focuses on day-to-day operation with larger account pools, richer import paths, clearer account health, and a more comfortable admin workflow.

Use only accounts and credentials you own or are authorized to manage. You are responsible for complying with Kiro, AWS, Amazon, and identity-provider terms.

## What This Project Does

- Exposes Claude-compatible, OpenAI-compatible, and Responses-compatible API endpoints.
- Routes requests across a Kiro account pool with refresh, failover, statistics, and per-account status.
- Provides a Web admin panel at `/admin` for adding accounts, testing accounts, refreshing quota/model metadata, and viewing request history.
- Supports multiple account credential sources, including Kiro OAuth flows, local cache import, credential JSON import, browser cookie import, and Kiro API key import.
- Keeps runtime data in `data/config.json` so Docker, local source builds, and hosted deployments can all persist the same account pool.

## Compared With zsecducna/Kiro-Go

| Area | Upstream focus | Enhanced branch |
| --- | --- | --- |
| Account import | Core Kiro login/import flows | Adds Microsoft / Entra ID SSO, Google/GitHub hosted login, Kiro API Key import, broader credential JSON compatibility, and local-cache focused workflows |
| Account health | Basic enabled/disabled and refresh state | Tracks callable accounts, suspended/banned/quota/auth/profile/sync states, real quota sync result, token health, and last failure reason |
| Account operations | Single-account management plus basic bulk actions | Adds batch test, batch model refresh, better account dedupe, created time display, region edit/detect, weight, proxy, and overage controls |
| API compatibility | Claude and OpenAI Chat Completions | Adds OpenAI Responses API support, `/v1/stats`, model aggregation, optional proxy API keys with per-key counters/limits, and thinking-mode formatting |
| Logs | Request log view | Separates client source from Kiro upstream endpoint, records account snapshots, shows clearer failure categories, and improves table scanning |
| Admin UI | Functional admin panel | Redesigned add-account cards, colored auth icons, sticky top navigation/stats/list controls for large account pools, polished stats popover, and bilingual copy |
| Region/profile handling | Region-bound account refresh | Adds account-level region visibility, region detection, profile ARN preservation, and configurable profile-region probing |
| Safety hygiene | Runtime config persistence | Ignores local `data/` and `auth-output/`, guards external IdP token endpoints, and blocks common duplicate/import mistakes |

## Key Features

- API endpoints:
  - Claude Messages: `/v1/messages`
  - Claude Count Tokens: `/v1/messages/count_tokens`
  - OpenAI Chat Completions: `/v1/chat/completions`
  - OpenAI Responses: `/v1/responses`
  - Models: `/v1/models`
  - Runtime stats: `/v1/stats`
- Account import methods:
  - AWS Builder ID
  - IAM Identity Center / Enterprise SSO
  - Microsoft 365 / Entra ID hosted SSO
  - Google/GitHub hosted Kiro login
  - Kiro API Key (`ksk_...`)
  - SSO token (`x-amz-sso_authn`)
  - Kiro IDE local cache
  - Kiro Account Manager / credential JSON
  - Kiro web cookie refresh token
- Account-pool management:
  - Round-robin routing with failover
  - Per-account enable/disable, delete, refresh, model refresh, and test
  - Batch enable, disable, refresh, model refresh, test, and delete
  - Per-account proxy URL and global outbound proxy
  - Per-account weight for priority routing
  - Quota/usage/credit sync, overage status, and subscription metadata
- Admin experience:
  - Sticky header, stats dashboard, list title, batch toolbar, search, and filter controls
  - Callable accounts shown as `available/total` with total credits
  - Hover summary for account availability, token state, quota state, and sync state
  - Request logs with source endpoint, upstream endpoint, model, account, tokens, latency, credits, and categorized error detail
  - Chinese and English UI

## Quick Start

### Docker Compose

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
mkdir -p data
docker compose up -d --build
```

Open `http://127.0.0.1:8080/admin`.

The Compose file publishes the Microsoft / Kiro hosted SSO callback port as `127.0.0.1:3128:3128`, so browser callback login works from the Docker host while staying off external interfaces.

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

### Build From Source

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
go build -o kiro-go .
CONFIG_PATH=data/config.json ADMIN_PASSWORD=change_this_password ./kiro-go
```

## Basic Usage

1. Open `http://127.0.0.1:8080/admin`.
2. Log in with the admin password. If you did not set one, the initial default is `changeme`.
3. Add one or more Kiro accounts from the account import dialog.
4. Refresh account usage and models.
5. Send API requests to the compatible endpoints.

Claude-compatible request:

```bash
curl http://127.0.0.1:8080/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4.5","max_tokens":1024,"messages":[{"role":"user","content":"Hello!"}]}'
```

OpenAI Chat Completions-compatible request:

```bash
curl http://127.0.0.1:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer any" \
  -d '{"model":"claude-sonnet-4.5","messages":[{"role":"user","content":"Hello!"}]}'
```

OpenAI Responses-compatible request:

```bash
curl http://127.0.0.1:8080/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer any" \
  -d '{"model":"claude-sonnet-4.5","input":"Hello!"}'
```

## Configuration

Runtime configuration is stored in `data/config.json` by default. Mount or back up `/app/data` when running in Docker.

| Variable | Description | Default |
| --- | --- | --- |
| `CONFIG_PATH` | Config file path | `data/config.json` |
| `ADMIN_PASSWORD` | Admin panel password override | config file value |
| `LOG_LEVEL` | Logger level, for example `debug`, `info`, `warn`, `error` | `info` |
| `KIRO_SSO_CALLBACK_BIND` | Bind address for the temporary hosted-SSO callback listener | loopback only |
| `KIRO_PROFILE_REGIONS` | Comma-separated profile region probe list for onboarding additional Kiro profile regions | built-in region list |

## API Authentication

The admin panel can create multiple proxy API keys. When API key verification is enabled, clients must send either:

- `Authorization: Bearer <key>`
- `X-Api-Key: <key>`

Each key can have independent enabled state, token/credit limits, usage counters, and last-used time.

## Thinking Mode

Append the configured suffix, default `-thinking`, to a model name, for example `claude-sonnet-4.5-thinking`.

Claude-compatible requests that include a top-level `thinking` config such as `{"type":"enabled","budget_tokens":2048}` or `{"type":"adaptive"}` also enable thinking mode automatically. Output format can be configured in the admin panel under Settings.

## Outbound Proxy

You can configure a global outbound proxy in Settings or set a per-account proxy in the account detail panel. SOCKS5 and HTTP proxies are supported. Settings apply without restarting the service.

## Deployment Notes

- Always set a strong `ADMIN_PASSWORD` before exposing the service.
- Enable proxy API key verification when the `/v1/*` endpoints are reachable by other machines.
- Do not commit `data/`, `auth-output/`, exported credential JSON files, or browser cache dumps.
- Keep the SSO callback port `3128` bound to `127.0.0.1` unless you know exactly why it must be exposed.
- Treat refresh tokens, access tokens, cookies, Kiro API keys, and exported account JSON as secrets.

## Updating From Upstream

This fork keeps `upstream` as `zsecducna/Kiro-Go`. A typical maintenance flow is:

```bash
git fetch upstream
git checkout codex-enhanced
git merge upstream/feat/azure-tenant-sso
```

Resolve conflicts carefully, then run tests and rebuild the Docker image.

## Disclaimer

For educational and research purposes only. This project is not affiliated with Amazon, AWS, Kiro, Microsoft, Google, or GitHub. Users are responsible for complying with applicable terms of service, identity-provider policies, and laws. Use at your own risk.

## License

[MIT](LICENSE)
