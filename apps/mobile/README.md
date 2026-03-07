# apps/mobile

Goyais 移动端（Tauri Mobile + Vue 复用）。

移动端通过 Vite alias 将 `@/*` 映射到 `apps/desktop/src/*`，以复用现有业务层与界面实现。

## 环境变量

- `VITE_HUB_BASE_URL`：移动端控制面 Hub 地址（必填，建议 `https://`）。
- `VITE_REQUIRE_HTTPS_HUB`：是否强制要求 HTTPS（默认移动端 release 为 `true`）。
- `VITE_ALLOW_INSECURE_HUB`：仅开发调试时允许 `http://` Hub。

## 启动 Web 调试

```bash
cd apps/mobile
pnpm install
pnpm dev:web
```

## 启动 iOS 调试

```bash
cd apps/mobile
pnpm dev:ios
```

## 启动 Android 调试

```bash
cd apps/mobile
pnpm dev:android
```

## 质量检查

```bash
cd apps/mobile
pnpm lint
pnpm test
pnpm e2e:smoke
```
