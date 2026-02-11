# java_server

Goyais Java Server v0.1 design-phase bootstrap.

## Modules

- `bom`: dependency versions.
- `parent`: shared Maven plugin and Java settings.
- `contract-api`: API DTO and common response models.
- `kernel-*`: shared kernel capabilities (core/web/security/mybatis).
- `capability-*`: cache/event/messaging/storage abstractions.
- `domain`, `application`, `infra-mybatis`, `adapter-rest`.
- `app-api-server`: business API entry.
- `app-auth-server`: auth server entry.

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
