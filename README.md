# Kiro-Go Enhanced

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

Kiro-Go Enhanced is a local or self-hosted Kiro account gateway. It turns authorized Kiro accounts into Claude, OpenAI Chat Completions, and OpenAI Responses compatible API endpoints, and provides a Web admin panel for account import, quota visibility, model discovery, request logs, and account-pool operations.

[English](README.md) | [中文](README_CN.md)

This repository uses [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) as its upstream source. The current branch is maintained as an operations-focused edition for people who need to run, observe, and maintain a larger Kiro account pool from one dashboard.

Use only accounts and credentials you own or are authorized to manage. You are responsible for complying with Kiro, AWS, Amazon, and identity-provider terms.

## Project Focus

- **Compatible API gateway**: expose Kiro-backed model access through familiar Claude and OpenAI-style `/v1/*` endpoints.
- **Account-pool operations**: manage many accounts from one place, with enable/disable, refresh, test, delete, weight, proxy, region, and overage controls.
- **Credential import hub**: add accounts from hosted OAuth, SSO token, local Kiro cache, credential JSON, browser cookie refresh token, or Kiro API Key.
- **Real operational visibility**: see callable accounts, total accounts, credits, quota state, token state, refresh failures, request success/failure counts, latency, and model metadata.
- **Large-list admin workflow**: sticky navigation, sticky stats, sticky list controls, batch actions, search, filters, bilingual UI, and clearer account status labels.
- **Deployment hygiene**: runtime secrets live in local `data/`, exported credentials are ignored by git/docker, and risky external IdP endpoints are validated before refresh tokens are posted.

## Main Capabilities

### API Compatibility

- Claude Messages: `/v1/messages`
- Claude Count Tokens: `/v1/messages/count_tokens`
- OpenAI Chat Completions: `/v1/chat/completions`
- OpenAI Responses: `/v1/responses`
- Model list: `/v1/models`
- Runtime statistics: `/v1/stats`

### Account Import Methods

- AWS Builder ID
- IAM Identity Center / Enterprise SSO
- Microsoft 365 / Entra ID hosted SSO
- Google/GitHub hosted Kiro login
- Kiro API Key (`ksk_...`)
- Browser `x-amz-sso_authn` SSO token
- Kiro IDE local cache
- Kiro Account Manager / credential JSON
- Kiro web cookie RefreshToken

### Account Pool Management

- Round-robin routing with request failover
- Auto token refresh and refresh failure classification
- Per-account enable/disable, delete, refresh, model refresh, and test
- Batch enable, disable, refresh, model refresh, test, and delete
- Per-account routing weight
- Per-account proxy URL plus global outbound proxy
- Region display, manual region edit, and region detection
- Subscription, quota, credits, trial, overage, and model cache metadata
- Duplicate import protection for common credential shapes

### Admin Dashboard

- Callable accounts displayed as `available/total`
- Total credits shown in the account stats card
- Hover summary for availability, disabled/banned/suspended state, token health, quota state, profile errors, and sync failures
- Request log viewer with source endpoint, Kiro upstream endpoint, model, account, tokens, latency, credits, and categorized details
- Add-account dialog with dedicated import cards and provider icons
- Sticky top bar, stats dashboard, account list header, batch toolbar, search, and filter controls
- Chinese and English interface

### Security-Oriented Controls

- `data/`, `auth-output/`, local logs, and exported credentials are ignored by git and docker context.
- External IdP issuer/token endpoints must be HTTPS and match the supported allow-list before refresh tokens are sent.
- Credential import request bodies are size-limited.
- Hosted SSO callback listener is loopback-only by default and time-limited.
- Proxy API keys can be required for all `/v1/*` client requests.
- Proxy API keys support enabled state, token/credit limits, usage counters, and last-used time.

## Quick Start

### Docker Compose

```bash
git clone -b codex-enhanced https://github.com/xiiyioozzz/Kiro-Go.git
cd Kiro-Go
mkdir -p data
docker compose up -d --build
```

Open `http://127.0.0.1:8080/admin`.

The Compose file publishes the hosted SSO callback port as `127.0.0.1:3128:3128`, so browser callback login works from the Docker host while staying off external interfaces.

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
  -e KIRO_SSO_CALLBACK_PORTS=3128 \
  -e KIRO_SOCIAL_CALLBACK_BIND=0.0.0.0 \
  -e KIRO_SOCIAL_CALLBACK_PORTS=3128 \
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

## First Run

1. Open `http://127.0.0.1:8080/admin`.
2. Log in with the admin password. If you did not set one, the initial default is `changeme`.
3. Add one or more Kiro accounts from the add-account dialog.
4. Refresh account usage and model metadata.
5. Use the `/v1/*` endpoints from your IDE, CLI, script, or compatible client.

## API Examples

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
| `KIRO_SSO_CALLBACK_BIND` | Bind address for the temporary Enterprise/Microsoft SSO callback listener | loopback only |
| `KIRO_SSO_CALLBACK_PORTS` | Comma-separated Enterprise/Microsoft SSO callback ports; publish every listed host port in Docker | `3128` |
| `KIRO_SOCIAL_CALLBACK_BIND` | Bind address for the temporary Google/GitHub Social callback listener | loopback only |
| `KIRO_SOCIAL_CALLBACK_PORTS` | Comma-separated Google/GitHub Social callback ports; publish every listed host port in Docker | `3128` |
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
- Do not commit `data/`, `auth-output/`, exported credential JSON files, browser cache dumps, or local logs.
- Keep the SSO callback port `3128` bound to `127.0.0.1` unless you know exactly why it must be exposed.
- Treat refresh tokens, access tokens, cookies, Kiro API keys, and exported account JSON as secrets.

## Upstream Reference

This repository keeps [Quorinex/Kiro-Go](https://github.com/Quorinex/Kiro-Go) as the upstream reference. A typical maintenance flow is:

```bash
git fetch upstream
git checkout codex-enhanced
git merge upstream/main
```

Resolve conflicts carefully, then run tests and rebuild the Docker image.

## Disclaimer

For educational and research purposes only. This project is not affiliated with Amazon, AWS, Kiro, Microsoft, Google, or GitHub. Users are responsible for complying with applicable terms of service, identity-provider policies, and laws. Use at your own risk.

## License

[MIT](LICENSE)
