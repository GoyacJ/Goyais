/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>MyBatisPlus implementation of share repository and permission checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.share.ShareRepository;
import com.ysmjjsy.goyais.application.share.ShareResourceScope;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Share;
import com.ysmjjsy.goyais.contract.api.common.ShareCreateRequest;
import com.ysmjjsy.goyais.infra.mybatis.entity.AssetEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.CommandEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.ShareEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.AssetEntityMapper;
import com.ysmjjsy.goyais.infra.mybatis.mapper.CommandEntityMapper;
import com.ysmjjsy.goyais.infra.mybatis.mapper.ShareEntityMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.io.IOException;
import java.time.Instant;
import java.util.Arrays;
import java.util.List;
import java.util.Locale;
import java.util.Set;
import java.util.UUID;
import org.springframework.stereotype.Repository;

/**
 * Persists share grants in acl_entries and resolves owner/permission checks.
 */
@Repository
public final class MybatisShareRepository implements ShareRepository {
    private final ShareEntityMapper shareMapper;
    private final CommandEntityMapper commandMapper;
    private final AssetEntityMapper assetMapper;
    private final ObjectMapper objectMapper;

    /**
     * Creates repository with share, command, and asset mappers.
     * @param shareMapper TODO
     * @param commandMapper TODO
     * @param assetMapper TODO
     * @param objectMapper TODO
     */
    public MybatisShareRepository(
            ShareEntityMapper shareMapper,
            CommandEntityMapper commandMapper,
            AssetEntityMapper assetMapper,
            ObjectMapper objectMapper
    ) {
        this.shareMapper = shareMapper;
        this.commandMapper = commandMapper;
        this.assetMapper = assetMapper;
        this.objectMapper = objectMapper;
    }

    /**
     * Persists one share row and returns mapped share resource.
     * @param request TODO
     * @param context TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public Share create(ShareCreateRequest request, ExecutionContext context, Instant now) {
        ShareEntity entity = new ShareEntity();
        entity.id = UUID.randomUUID().toString();
        entity.tenantId = context.tenantId();
        entity.workspaceId = context.workspaceId();
        entity.resourceType = request.resourceType();
        entity.resourceId = request.resourceId();
        entity.subjectType = request.subjectType();
        entity.subjectId = request.subjectId();
        entity.permissionsJson = writePermissions(request.permissions());
        entity.expiresAt = request.expiresAt();
        entity.createdBy = context.userId();
        entity.createdAt = now == null ? Instant.now() : now;
        shareMapper.insert(entity);
        return toResource(entity);
    }

    /**
     * Returns share list for scope with deterministic ordering.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    @Override
    public List<Share> list(ExecutionContext context, int page, int pageSize) {
        int normalizedPage = Math.max(page, 1);
        int normalizedPageSize = pageSize <= 0 ? 20 : Math.min(pageSize, 200);
        int offset = (normalizedPage - 1) * normalizedPageSize;

        return shareMapper.selectPage(context.tenantId(), context.workspaceId(), normalizedPageSize, offset).stream()
                .map(this::toResource)
                .toList();
    }

    /**
     * Returns share count in current tenant/workspace scope.
     * @param context TODO
     * @return TODO
     */
    @Override
    public long count(ExecutionContext context) {
        return shareMapper.countByScope(context.tenantId(), context.workspaceId());
    }

    /**
     * Deletes one share created by current user.
     * @param shareId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public boolean deleteByIdAndCreator(String shareId, ExecutionContext context) {
        return shareMapper.deleteByIdAndCreator(
                shareId,
                context.tenantId(),
                context.workspaceId(),
                context.userId()
        ) > 0;
    }

    /**
     * Returns resource scope projection for command or asset resources.
     * @param resourceType TODO
     * @param resourceId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public ShareResourceScope findResourceScope(String resourceType, String resourceId, ExecutionContext context) {
        String normalized = resourceType == null ? "" : resourceType.trim().toLowerCase(Locale.ROOT);
        return switch (normalized) {
            case "command" -> toScope(commandMapper.selectByIdInScope(resourceId, context.tenantId(), context.workspaceId()));
            case "asset" -> toScope(assetMapper.selectByIdInScope(resourceId, context.tenantId(), context.workspaceId()));
            default -> null;
        };
    }

    /**
     * Returns true when user/roles have requested permission on resource.
     * @param resourceType TODO
     * @param resourceId TODO
     * @param context TODO
     * @param permission TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public boolean hasResourcePermission(
            String resourceType,
            String resourceId,
            ExecutionContext context,
            Permission permission,
            Instant now
    ) {
        String normalizedType = resourceType == null ? "" : resourceType.trim().toLowerCase(Locale.ROOT);
        String permissionValue = permission == null ? Permission.READ.name() : permission.name();
        Instant ts = now == null ? Instant.now() : now;

        boolean userAllowed = shareMapper.hasPermissionForUser(
                context.tenantId(),
                context.workspaceId(),
                normalizedType,
                resourceId,
                context.userId(),
                permissionValue,
                ts
        );
        if (userAllowed) {
            return true;
        }

        List<String> roles = context.roles() == null ? List.of() : context.roles().stream()
                .filter(role -> role != null && !role.isBlank())
                .map(role -> role.trim().toLowerCase(Locale.ROOT))
                .distinct()
                .toList();
        if (roles.isEmpty()) {
            return false;
        }

        return shareMapper.hasPermissionForRoles(
                context.tenantId(),
                context.workspaceId(),
                normalizedType,
                resourceId,
                roles,
                permissionValue,
                ts
        );
    }

    private ShareResourceScope toScope(CommandEntity entity) {
        if (entity == null) {
            return null;
        }
        return new ShareResourceScope(
                "command",
                entity.getId(),
                entity.getTenantId(),
                entity.getWorkspaceId(),
                entity.getOwnerId(),
                entity.getVisibility()
        );
    }

    private ShareResourceScope toScope(AssetEntity entity) {
        if (entity == null) {
            return null;
        }
        return new ShareResourceScope(
                "asset",
                entity.id,
                entity.tenantId,
                entity.workspaceId,
                entity.ownerId,
                entity.visibility
        );
    }

    private Share toResource(ShareEntity entity) {
        return new Share(
                entity.id,
                entity.tenantId,
                entity.workspaceId,
                entity.resourceType,
                entity.resourceId,
                entity.subjectType,
                entity.subjectId,
                readPermissions(entity.permissionsJson),
                entity.expiresAt,
                entity.createdBy,
                entity.createdAt
        );
    }

    private String writePermissions(Set<Permission> permissions) {
        Permission[] ordered = permissions == null ? new Permission[0] : permissions.toArray(new Permission[0]);
        Arrays.sort(ordered);
        try {
            return objectMapper.writeValueAsString(ordered);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize share permissions", ex);
        }
    }

    private Set<Permission> readPermissions(String value) {
        if (value == null || value.isBlank()) {
            return Set.of();
        }
        try {
            List<String> raw = objectMapper.readValue(value, new TypeReference<>() {
            });
            return raw.stream()
                    .map(item -> {
                        try {
                            return Permission.valueOf(item);
                        } catch (IllegalArgumentException ex) {
                            return null;
                        }
                    })
                    .filter(item -> item != null)
                    .collect(java.util.stream.Collectors.toCollection(java.util.LinkedHashSet::new));
        } catch (IOException ex) {
            throw new IllegalStateException("failed to deserialize share permissions", ex);
        }
    }
}
