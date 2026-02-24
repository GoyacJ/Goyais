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

在本机需要先执行（按当前目标三元组）：

```bash
TARGET_TRIPLE="$(rustc -vV | awk '/^host:/ {print $2}')"
scripts/release/build-hub-sidecar.sh "$TARGET_TRIPLE"
scripts/release/build-worker-sidecar.sh "$TARGET_TRIPLE"
```

## 质量检查

```bash
cd apps/desktop
pnpm lint
pnpm test
```
