# 控制台优化方案 v3（已落地 Phase 0）

## 已定产品决策

| 项 | 方案 |
|----|------|
| Slogan | **Simple. Cheap. Fast.** / **简单 · 便宜 · 高效** |
| Slogan 动画 | **每次登录成功**播完整居中动画，之后仅顶栏紧凑条；刷新页面不重复 |
| 首页 | **不设独立首页**，`/` = Usage |
| 语言 | 顶栏 **EN \| 中文** 全局切换，`localStorage.sub2api_locale` |
| 卡片动效 | **仅页面 mount 一次**整卡弹性跳动（锤子解锁图标风格），无点击/hover 动画 |

## Phase 0 已实现

- `src/lib/i18n/` — 中英字典、`LocaleProvider`、`useT`
- `src/components/locale-toggle.tsx`
- `src/components/brand/slogan-stage.tsx` — 每日首次 hero
- `src/components/ui/ripple-card.tsx`
- `src/lib/brand.ts` — 登录后 Slogan 触发（sessionStorage）
- `ConsoleShell` — Slogan + 语言切换 + Teal 主题
- `usage-page` — i18n + RippleCard
- `login-form` / `auth-dialog` — i18n

## localStorage

| Key | 说明 |
|-----|------|
| `sub2api_locale` | `en`（默认）\| `zh` |
| `sub2api_slogan_after_login` | sessionStorage：登录后跨页时触发 Slogan，播放后清除 |

## Phase 1（已完成）

- 全站页面 i18n：keys / topup / billing / subscription / chat / logs / profile / payment / error
- 全站主要卡片 `RippleCard`（进场水波一次）
- `/terms` 中文协议 `terms-content-zh.tsx`
- 顶栏新增 **Logs** 导航
- 流水类型 `transaction-labels` 支持 `t()`

## Phase 2（已完成）

### 后端 `sub2api-go`

- `GET /dashboard/usage-daily?scope=account&days=7|14|30` — 账户级日消费
- `GET /dashboard/usage/summary` — 本月消费与请求数
- `GET /dashboard/usage/by-model?days=` — 按模型聚合
- `GET /dashboard/usage/export?month=YYYY-MM` — CSV 导出
- `GET /dashboard/payments` — 充值流水（ledger topup）
- `AggregateConsumeByDay` 计入 `chat_consume` / `api_consume` / `consume`
- SQLite `account_ledger` 为主数据源；Redis 无 SQLite 时回退扫描 `tx:*`

### 前端 `dashboard`

- Usage：账户趋势图（7/14/30）、本月统计卡、按模型表、CSV 导出、保留按密钥图
- Top-up：充值记录表
- i18n：`usage.*` / `topup.paymentHistory` 等

## Phase UI（codesome 布局）

- 左侧固定导航栏（图标 + 文案，Teal 高亮当前项）
- 顶栏仅显示页面标题 + 登录区（移动端汉堡菜单）
- `StatTile` / `PanelCard` 统一指标卡与大面板排版
- `RippleCard` 整卡弹性跳动（mount 一次，`transform-origin: bottom`）

## Phase 2b（待做）

- API 错误信息中文化（后端 `Accept-Language`）
