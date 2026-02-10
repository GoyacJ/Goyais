# verify_single_binary.sh 使用说明

脚本路径：`scripts/verify_single_binary.sh`

## 默认行为

- 默认基地址：`GOYAIS_VERIFY_BASE_URL=http://127.0.0.1:8080`
- 默认流程：`make build` -> 自动探测二进制 -> 启动服务 -> 验收检查

## 可选环境变量

- `GOYAIS_VERIFY_BASE_URL`：服务访问地址
- `GOYAIS_BINARY_PATH`：显式指定可执行文件路径（可覆盖自动探测）
- `GOYAIS_START_CMD`：显式指定启动命令（可覆盖默认直接执行二进制）

## 退出码

- `0`：全部通过
- `1`：任一验收项失败
- `2`：未找到可执行构建产物
- `3`：`make build` 失败
- `4`：服务启动或健康检查超时

## 最小执行示例

```bash
bash .agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh
```

## 指定二进制示例

```bash
GOYAIS_BINARY_PATH=./bin/goyais \
bash .agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh
```
