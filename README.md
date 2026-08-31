<p align="right">
  <strong>中文</strong> | <a href="./README.en.md">English</a>
</p>

<div align="center">

# RingRouter

_✦ 一个网关，所有供应商 —— 多协议进、多渠道出、故障自愈的 LLM API 网关 ✦_

</div>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-AGPL--3.0-blue" alt="license"></a>
  <img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go" alt="go">
  <img src="https://img.shields.io/badge/React-19-61DAFB?logo=react" alt="react">
  <img src="https://img.shields.io/badge/docker-compose-ready-2496ED?logo=docker" alt="docker">
</p>

<p align="center">
  <a href="#功能">功能</a> ·
  <a href="#快速开始">快速开始</a> ·
  <a href="#环境变量">环境变量</a> ·
  <a href="#使用方法">使用方法</a> ·
  <a href="#与-one-api-的差异">与 one-api 的差异</a> ·
  <a href="#开发">开发</a>
</p>

> [!NOTE]
> 本项目为自部署网关，使用者须遵循各上游供应商的服务条款与所在地区法律法规。

## 功能

**协议层**
- [x] 四种入站协议，任意进、任意出（统一中间格式互转）：OpenAI Chat Completions / OpenAI Responses / Anthropic Messages / Google Gemini `generateContent`
- [x] 流式（SSE）与非流式均支持；同协议流式原样透传，跨协议自动翻译
- [x] `GET /v1/models` 跨渠道聚合模型列表

**路由层**
- [x] 多渠道：按模型匹配 → 优先级排序 → 逐个故障转移
- [x] 渠道密钥 AES-GCM 加密落库，管理界面永不回显
- [x] 模型映射（客户端模型名 → 上游模型名，JSON 配置）
- [x] 渠道缓存：进程内 30s 快照；可选 Redis 共享快照（跨实例一致，写后即失效）

**用户与计费**
- [x] 分组：`name / uuid / metadata / ratio`（计费倍率），渠道可属多个分组（逗号分隔），改名自动级联
- [x] 计划与订阅：计划绑定配额+分组+周期，分配即生效；订阅快照式记录，惰性过期
- [x] API 密钥（`sk-rr-` 前缀）：创建时一次性展示，此后仅掩码
- [x] 请求日志：模型 / token 数 / 耗时 / 渠道 / IP 异步落库，个人与管理员双视角查询

**认证**
- [x] 用户名+密码登录，邮箱验证码重置密码（60s 冷却、5 次尝试上限、单次使用）
- [x] 通行密钥（WebAuthn / Passkey）：免密登录、设备内注册、discoverable 流程
- [x] Cloudflare Turnstile 人机验证（可选）
- [x] 管理密钥（`ADMIN_KEY`）可直接换取管理会话

**管理面**
- [x] 4 步安装向导（站点信息 → SMTP → 通行密钥 → 使用模式）
- [x] 渠道 / 用户 / 分组 / 计划 / 订阅 / 模型目录 / 日志 / 系统设置全功能管理页
- [x] 公开模型广场（`/models`）：无需登录浏览可用模型与分组倍率
- [x] Playground：内置流式对话调试
- [x] 中 / 繁（台 / 港） / 英四语言，深色模式，单二进制部署（前端嵌入）

**暂未实现**（欢迎 PR）：负载均衡随机分流、兑换码、邀请奖励、OAuth2 登录

## 快速开始

### Docker Compose（推荐，PostgreSQL + Redis）

```shell
git clone https://github.com/RingKoAI/RingRouter.git
cd RingRouter
cp .env.example .env          # 修改 ADMIN_KEY / JWT_SECRET / 数据库密码
docker compose up -d --build
```

打开 `http://localhost:3000`，按引导完成安装（创建管理员账号）。

### Docker 单容器（SQLite 轻量模式）

先从源码构建镜像，再运行：

```shell
docker build -t ringrouter .
docker run --name ringrouter -d --restart always -p 3000:3000 \
  -e ADMIN_KEY=change-me -e JWT_SECRET=change-me-too \
  -e DB_TYPE=sqlite -v /data/ringrouter:/app/data \
  ringrouter
```

### 源码构建

```shell
git clone https://github.com/RingKoAI/RingRouter.git
cd RingRouter

# 前端
cd web && pnpm install && pnpm build && cd ..

# 后端（嵌入 web/dist，产出单二进制）
go build -o ringrouter .
ADMIN_KEY=change-me ./ringrouter
```

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `PORT` | 监听端口 | `3000` |
| `DB_TYPE` | `postgres` / `mysql` / `sqlite` | `postgres` |
| `DB_DSN` | PG / MySQL 连接串 | — |
| `DB_PATH` | SQLite 文件路径（仅 sqlite） | `data/ringrouter.db` |
| `ADMIN_KEY` | 管理引导密钥，可换取管理会话 | 随机 |
| `JWT_SECRET` | 密钥密封与签名盐（AES-GCM 派生）；留空时自动生成 256bit 随机值并持久化到 `data/.instance_secret`（0600） | 自动生成并持久化 |
| `ENCRYPTION_KEY` | 独立加密密钥（hex 32 字节），优先于 `JWT_SECRET` 派生 | — |
| `REDIS_CONN_STRING` | `redis://[user[:pass]@]host:port/db`，设置即启用共享缓存 | 未启用 |
| `REDIS_ENABLED` 等分离变量 | `REDIS_ENABLED=true` + `REDIS_ADDR/PASSWORD/DB` 的替代写法 | 未启用 |
| `TRUSTED_PROXIES` | 反向代理 CIDR 列表（逗号分隔）；仅来自这些地址的连接信任 `X-Forwarded-For`/`X-Real-IP`。`*` 信任任意，`none` 完全禁用 | 仅 loopback |
| `CHANNEL_ALLOW_PRIVATE_ADDR` | 渠道 base_url / SMTP 测试允许内网地址；`false` 开启 SSRF 加固（拒绝环回/私网/链路本地，含出站重定向逐跳校验） | `true` |
| `OPENAI_API_KEY` / `OPENAI_BASE_URL` | 无数据库渠道时的兜底上游（可选） | — |
| `ANNOUNCEMENT` | 首次启动播种公告（可选） | — |
| `TURNSTILE_SITEKEY` / `TURNSTILE_SECRET` | Cloudflare Turnstile（登录/注册/安装/SMTP 测试） | 未启用 |
| `RATE_LIMIT_API` / `RATE_LIMIT_WEB` / `RATE_LIMIT_CRITICAL` | 每 IP 滑动窗口限额（网关 / 管理面 / 登录注册等敏感端点，`0` 关闭） | `480` / `240` / `20` |

> [!IMPORTANT]
> - 生产部署务必固定 `JWT_SECRET`（或保留自动持久化的 `data/.instance_secret` 且勿删除），并启用 HTTPS；compose 模板默认仅绑定 `127.0.0.1`，由前置反代对外服务。
> - 直接对外（无反代）部署时，转发头会被伪造以绕过限流；保持默认「仅信任 loopback」或按拓扑配置 `TRUSTED_PROXIES`。

## 使用方法

1. **安装向导**：首次访问自动进入 `/setup`，创建管理员、可选配置 SMTP 与通行密钥
2. **添加渠道**：控制台 → 渠道管理 → 填写上游协议（openai / anthropic / google）、地址与密钥、模型列表、分组与优先级
3. **创建密钥**：控制台 → API 密钥 → 生成 `sk-rr-…`（仅显示一次）
4. **调用网关**（任选一种协议，任意客户端兼容）：

```bash
# OpenAI 协议
curl http://localhost:3000/v1/chat/completions \
  -H "Authorization: Bearer sk-rr-xxxx" -H "Content-Type: application/json" \
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}'

# Anthropic 协议（同一密钥）
curl http://localhost:3000/v1/messages \
  -H "x-api-key: sk-rr-xxxx" -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-sonnet-4-5","max_tokens":64,"messages":[{"role":"user","content":"hi"}]}'

# Gemini 协议（同一密钥）
curl "http://localhost:3000/v1beta/models/gemini-2.0-flash:generateContent" \
  -H "x-goog-api-key: sk-rr-xxxx" \
  -d '{"contents":[{"parts":[{"text":"hi"}]}]}'
```

5. **公开模型广场**：`/models` 无需登录，浏览全部可用模型与分组倍率

## 与 one-api 的差异

RingRouter 在设计上参考并致敬 [one-api](https://github.com/songquanpeng/one-api)（分组倍率、渠道优先级、Redis 可选缓存等语义保持一致），主要差异：

| 维度 | RingRouter | one-api |
|------|-----------|---------|
| 入站协议 | 四协议互转（含 Responses / Gemini） | OpenAI 兼容为主 |
| 分组 | 独立实体表（uuid / metadata / ratio）+ 渠道多分组 | 字符串约定 + 倍率配置 |
| 订阅 | 计划/订阅快照式生命周期 | 无（额度充值模型） |
| 认证 | 密码 + 邮箱验证码 + Passkey + Turnstile | 密码 + 邮箱 + 多 OAuth |
| 计费 | 计划分配 + 按次扣减（价格 × 分组倍率，$1=50万点） | 完整额度体系 |

## 开发

```bash
# 后端（Go 1.27）
go build ./... && go vet ./... && go test ./internal/...

# 前端（web/，pnpm）
cd web && pnpm install && pnpm dev   # Vite :5173，代理到 :3000
```

目录结构：`internal/` 后端（config / crypto / database / gateway / handler / inbound / middleware / model / provider / setting / turnstile / cache），`web/` 前端（React 19 + Vite + Tailwind v4）。界面翻译位于 `web/src/i18n/locales`（zh / zh-TW / zh-HK / en 四语言，新增文案必须四份同步）。

## License

[AGPL-3.0](./LICENSE)

本仓库不含任何 AGPL 许可的第三方代码；采用 AGPL-3.0 是为了保持开源网关生态的传染性开源要求。
