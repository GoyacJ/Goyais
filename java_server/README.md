# java_server

Goyais Java Server v0.1 design-phase bootstrap.

## Modules

- `bom`: dependency versions.
- `parent`: shared Maven plugin and Java settings.
- `contract-api`: API DTO and common response models.
- `kernel-*`: shared kernel capabilities (core/web/security/mybatis).
- `capability-*`: cache/event/messaging/storage abstractions.
- `domain`, `application`, `infra-mybatis`, `adapter-rest`.
- `app-api-server`: single runtime entry (resource + auth capability in one process).
- `app-auth-server`: reusable auth capability module (assembled by `app-api-server`).

## Docs

- `docs/development-spec.md`
- `docs/development-plan.md`
- `docs/api/openapi-java-draft.yaml`
- `docs/arch/*`
- `docs/acceptance.md`

## Quick Build

```bash
mvn -f java_server/pom.xml -DskipTests verify
```
