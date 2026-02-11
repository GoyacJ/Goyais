/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>MyBatisPlus implementation of asset repository with SQL data-permission filtering.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.asset.AssetRepository;
import com.ysmjjsy.goyais.contract.api.common.AclItem;
import com.ysmjjsy.goyais.contract.api.common.Asset;
import com.ysmjjsy.goyais.contract.api.common.AssetLineageEdge;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.infra.mybatis.entity.AssetEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.AssetLineageEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.AssetEntityMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionResolver;
import java.io.IOException;
import java.time.Instant;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import org.springframework.stereotype.Repository;

/**
 * Persists assets and executes permission-aware list/get queries.
 */
@Repository
public final class MybatisAssetRepository implements AssetRepository {
    private final AssetEntityMapper mapper;
    private final DataPermissionResolver dataPermissionResolver;
    private final ObjectMapper objectMapper;

    /**
     * Creates repository with mapper, data permission resolver, and JSON codec.
     * @param mapper TODO
     * @param dataPermissionResolver TODO
     * @param objectMapper TODO
     */
    public MybatisAssetRepository(
            AssetEntityMapper mapper,
            DataPermissionResolver dataPermissionResolver,
            ObjectMapper objectMapper
    ) {
        this.mapper = mapper;
        this.dataPermissionResolver = dataPermissionResolver;
        this.objectMapper = objectMapper;
    }

    /**
     * Persists one asset row and returns mapped resource.
     * @param context TODO
     * @param name TODO
     * @param type TODO
     * @param mime TODO
     * @param size TODO
     * @param hash TODO
     * @param uri TODO
     * @param visibility TODO
     * @param metadata TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public Asset create(
            ExecutionContext context,
            String name,
            String type,
            String mime,
            long size,
            String hash,
            String uri,
            Visibility visibility,
            Map<String, Object> metadata,
            Instant now
    ) {
        Instant ts = now == null ? Instant.now() : now;
        AssetEntity entity = new AssetEntity();
        entity.id = UUID.randomUUID().toString();
        entity.tenantId = context.tenantId();
        entity.workspaceId = context.workspaceId();
        entity.ownerId = context.userId();
        entity.visibility = (visibility == null ? Visibility.PRIVATE : visibility).name();
        entity.aclJson = "[]";
        entity.name = name;
        entity.type = type;
        entity.mime = mime;
        entity.size = size;
        entity.uri = uri;
        entity.hash = hash;
        entity.metadataJson = writeJson(metadata == null ? Map.of() : metadata);
        entity.status = "ready";
        entity.createdAt = ts;
        entity.updatedAt = ts;
        mapper.insert(entity);
        return toResource(entity);
    }

    /**
     * Returns one in-scope asset regardless of ACL read filters.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public Asset findByIdInScope(String assetId, ExecutionContext context) {
        AssetEntity entity = mapper.selectByIdInScope(assetId, context.tenantId(), context.workspaceId());
        return entity == null ? null : toResource(entity);
    }

    /**
     * Returns one readable asset by identifier, or null.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public Asset findReadableById(String assetId, ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        AssetEntity entity = mapper.selectReadableById(assetId, predicate, dp);
        return entity == null ? null : toResource(entity);
    }

    /**
     * Returns readable assets for requested page.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    @Override
    public List<Asset> listReadable(ExecutionContext context, int page, int pageSize) {
        int normalizedPage = Math.max(page, 1);
        int normalizedPageSize = pageSize <= 0 ? 20 : Math.min(pageSize, 200);
        int offset = (normalizedPage - 1) * normalizedPageSize;

        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return mapper.selectReadableList(predicate, dp, normalizedPageSize, offset).stream()
                .map(this::toResource)
                .toList();
    }

    /**
     * Returns readable asset count for current context.
     * @param context TODO
     * @return TODO
     */
    @Override
    public long countReadable(ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return mapper.countReadable(predicate, dp);
    }

    /**
     * Updates mutable asset fields and returns latest resource.
     * @param assetId TODO
     * @param context TODO
     * @param name TODO
     * @param visibility TODO
     * @param metadata TODO
     * @param metadataProvided TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public Asset update(
            String assetId,
            ExecutionContext context,
            String name,
            Visibility visibility,
            Map<String, Object> metadata,
            boolean metadataProvided,
            Instant now
    ) {
        mapper.updateMutable(
                assetId,
                context.tenantId(),
                context.workspaceId(),
                name,
                visibility == null ? null : visibility.name(),
                metadataProvided ? writeJson(metadata == null ? Map.of() : metadata) : null,
                metadataProvided,
                now == null ? Instant.now() : now
        );
        return findByIdInScope(assetId, context);
    }

    /**
     * Marks one asset as deleted and returns latest resource.
     * @param assetId TODO
     * @param context TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public Asset markDeleted(String assetId, ExecutionContext context, Instant now) {
        mapper.markDeleted(assetId, context.tenantId(), context.workspaceId(), now == null ? Instant.now() : now);
        return findByIdInScope(assetId, context);
    }

    /**
     * Returns true when user or roles have requested asset permission.
     * @param assetId TODO
     * @param context TODO
     * @param permission TODO
     * @param now TODO
     * @return TODO
     */
    @Override
    public boolean hasPermission(String assetId, ExecutionContext context, Permission permission, Instant now) {
        Instant ts = now == null ? Instant.now() : now;
        String permissionValue = permission == null ? Permission.READ.name() : permission.name();

        boolean userAllowed = mapper.hasPermissionForUser(
                context.tenantId(),
                context.workspaceId(),
                assetId,
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

        return mapper.hasPermissionForRoles(
                context.tenantId(),
                context.workspaceId(),
                assetId,
                roles,
                permissionValue,
                ts
        );
    }

    /**
     * Returns lineage edges for one asset identifier.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public List<AssetLineageEdge> listLineage(String assetId, ExecutionContext context) {
        return mapper.selectLineage(context.tenantId(), context.workspaceId(), assetId).stream()
                .map(this::toLineageEdge)
                .toList();
    }

    private DataPermissionContext toDataPermissionContext(ExecutionContext context) {
        return new DataPermissionContext(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                context.roles(),
                context.policyVersion(),
                "asset",
                Permission.READ.name()
        );
    }

    private Asset toResource(AssetEntity entity) {
        ResourceBase base = new ResourceBase(
                entity.id,
                entity.tenantId,
                entity.workspaceId,
                entity.ownerId,
                parseVisibility(entity.visibility),
                List.<AclItem>of(),
                entity.status,
                entity.createdAt,
                entity.updatedAt
        );

        return new Asset(
                base,
                entity.name,
                entity.type,
                entity.mime,
                entity.size == null ? 0L : entity.size,
                entity.hash,
                entity.uri,
                readJsonMap(entity.metadataJson)
        );
    }

    private AssetLineageEdge toLineageEdge(AssetLineageEntity entity) {
        return new AssetLineageEdge(
                entity.id,
                entity.sourceAssetId,
                entity.targetAssetId,
                entity.runId,
                entity.stepId,
                entity.relation,
                entity.createdAt
        );
    }

    private Visibility parseVisibility(String raw) {
        if (raw == null || raw.isBlank()) {
            return Visibility.PRIVATE;
        }
        try {
            return Visibility.valueOf(raw.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            return Visibility.PRIVATE;
        }
    }

    private String writeJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value == null ? Map.of() : value);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize asset metadata", ex);
        }
    }

    private Map<String, Object> readJsonMap(String value) {
        if (value == null || value.isBlank()) {
            return Map.of();
        }
        try {
            return objectMapper.readValue(value, new TypeReference<>() {
            });
        } catch (IOException ex) {
            throw new IllegalStateException("failed to deserialize asset metadata", ex);
        }
    }
}
