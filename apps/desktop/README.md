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

当前默认行为：`pnpm dev:desktop` / `pnpm tauri:dev` 会在每次启动前强制重建本机 Hub sidecar（`GOYAIS_FORCE_SIDECAR_REBUILD=1`）。

这样可以避免 Hub 重构后仍误用旧 sidecar 二进制，导致消息发送后 execution 长时间停留在 `pending`（界面表现为“正在准备下一步”无进展）。

若你要手动执行（或强制重建）：

```bash
TARGET_TRIPLE="$(rustc -vV | awk '/^host:/ {print $2}')"
scripts/release/build-hub-sidecar.sh "$TARGET_TRIPLE"

# 强制重建（忽略本地已有 sidecar）
GOYAIS_FORCE_SIDECAR_REBUILD=1 bash ../../scripts/release/ensure-local-sidecars.sh
```

## 排障：确认是否运行了旧 Hub sidecar

若出现“发送消息后一直卡住”，先检查当前运行二进制的 revision：

```bash
go version -m apps/desktop/src-tauri/target/debug/goyais-hub
git rev-parse HEAD
```

若两者 revision 不一致，先重启 `pnpm tauri:dev`（会自动强制重建 sidecar）。

## 质量检查

```bash
cd apps/desktop
pnpm lint
pnpm test
```
