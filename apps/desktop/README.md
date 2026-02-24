# apps/desktop

Goyais v0.4.0 Desktop 骨架（Tauri + Vue + TypeScript）。

## 启动 Web 调试

```bash
cd apps/desktop
pnpm install
pnpm dev --port 5173
```

## 启动 Tauri 调试

```bash
cd apps/desktop
pnpm tauri:dev
```

## Sidecar 产物准备（开箱即用包）

`Tauri` 在编译时会检查 `src-tauri/binaries` 下的 sidecar 文件是否存在。

默认情况下，`pnpm dev:desktop` / `pnpm tauri:dev` 会自动检查并按需生成本机 sidecar。

若你要手动执行（或强制重建）：

```bash
TARGET_TRIPLE="$(rustc -vV | awk '/^host:/ {print $2}')"
scripts/release/build-hub-sidecar.sh "$TARGET_TRIPLE"
scripts/release/build-worker-sidecar.sh "$TARGET_TRIPLE"

# 强制重建（忽略本地已有 sidecar）
GOYAIS_FORCE_SIDECAR_REBUILD=1 bash ../../scripts/release/ensure-local-sidecars.sh
```

## 质量检查

```bash
cd apps/desktop
pnpm lint
pnpm test
```
