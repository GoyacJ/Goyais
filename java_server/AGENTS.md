# java_server AGENTS (Implementation Phase)

当前模块已进入实现阶段（implementation phase），默认执行单应用拓扑。

## Runtime Topology (MUST)

- 默认模式：`single`（Auth + Resource 同进程）。
- 扩展模式：`resource-only`（仅资源服务器，外接独立授权中心）。
- 配置键：`GOYAIS_SECURITY_TOPOLOGY_MODE=single|resource-only`。

## Engineering Constraints (MUST)

- 继续对齐 `docs/prd.md` 与 `go_server/docs/*` 契约，不得破坏 `/api/v1` 同构语义。
- 所有副作用动作保持 command-first，domain sugar 必须映射到 command 并写审计。
- 动态权限必须基于 `policyVersion`，并支持 Redis 失效广播。
- 数据权限首期固定为行级 SQL 过滤，不允许业务层散落式权限分支。

## Comment and Quality Gate (MUST)

- 每个 Java 源码文件必须具备标准文件头。
- `public class/interface/enum/record` 与 `public` 方法/构造器必须提供 JavaDoc。
- 提交前至少执行：
  - `bash go_server/scripts/ci/source_header_check.sh`
  - `bash java_server/scripts/ci/java_javadoc_check.sh`
  - `mvn -f java_server/pom.xml -DskipTests verify`
