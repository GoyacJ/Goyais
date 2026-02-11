/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: MyBatisPlus implementation of workflow template repository with SQL permission filtering.
 */
package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.workflow.WorkflowTemplateRepository;
import com.ysmjjsy.goyais.contract.api.common.AclItem;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowTemplateEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.WorkflowTemplateEntityMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionResolver;
import java.io.IOException;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import org.springframework.stereotype.Repository;

/**
 * Persists workflow templates and applies row-level read filters in SQL queries.
 */
@Repository
public final class MybatisWorkflowTemplateRepository implements WorkflowTemplateRepository {
    private final WorkflowTemplateEntityMapper mapper;
    private final DataPermissionResolver dataPermissionResolver;
    private final ObjectMapper objectMapper;

    /**
     * Creates repository with template mapper, data permission resolver, and JSON codec.
     */
    public MybatisWorkflowTemplateRepository(
            WorkflowTemplateEntityMapper mapper,
            DataPermissionResolver dataPermissionResolver,
            ObjectMapper objectMapper
    ) {
        this.mapper = mapper;
        this.dataPermissionResolver = dataPermissionResolver;
        this.objectMapper = objectMapper;
    }

    /**
     * Persists one draft workflow template and returns mapped resource.
     */
    @Override
    public WorkflowTemplate createDraft(
            ExecutionContext context,
            String name,
            String description,
            Visibility visibility,
            Map<String, Object> graph,
            Map<String, Object> schemaInputs,
            Map<String, Object> schemaOutputs,
            Map<String, Object> uiState,
            Instant now
    ) {
        Instant ts = now == null ? Instant.now() : now;
        WorkflowTemplateEntity entity = new WorkflowTemplateEntity();
        entity.id = UUID.randomUUID().toString();
        entity.tenantId = context.tenantId();
        entity.workspaceId = context.workspaceId();
        entity.ownerId = context.userId();
        entity.visibility = (visibility == null ? Visibility.PRIVATE : visibility).name();
        entity.aclJson = "[]";
        entity.name = name;
        entity.description = description == null ? "" : description;
        entity.status = "draft";
        entity.currentVersion = 1;
        entity.graphJson = writeJson(graph == null ? Map.of() : graph);
        entity.schemaInputsJson = writeJson(schemaInputs == null ? Map.of() : schemaInputs);
        entity.schemaOutputsJson = writeJson(schemaOutputs == null ? Map.of() : schemaOutputs);
        entity.uiStateJson = writeJson(uiState == null ? Map.of() : uiState);
        entity.createdAt = ts;
        entity.updatedAt = ts;
        mapper.insert(entity);
        return toResource(entity);
    }

    /**
     * Returns one template in tenant/workspace scope without ACL read filtering.
     */
    @Override
    public WorkflowTemplate findByIdInScope(String templateId, ExecutionContext context) {
        WorkflowTemplateEntity entity = mapper.selectByIdInScope(templateId, context.tenantId(), context.workspaceId());
        return entity == null ? null : toResource(entity);
    }

    /**
     * Returns one readable template by id, or null when inaccessible.
     */
    @Override
    public WorkflowTemplate findReadableById(String templateId, ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        WorkflowTemplateEntity entity = mapper.selectReadableById(templateId, predicate, dp);
        return entity == null ? null : toResource(entity);
    }

    /**
     * Returns readable template list with deterministic descending order.
     */
    @Override
    public List<WorkflowTemplate> listReadable(ExecutionContext context, int page, int pageSize) {
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
     * Returns count of readable templates for current context.
     */
    @Override
    public long countReadable(ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return mapper.countReadable(predicate, dp);
    }

    /**
     * Updates template graph/ui-state payload and returns latest template resource.
     */
    @Override
    public WorkflowTemplate patch(
            String templateId,
            ExecutionContext context,
            Map<String, Object> graph,
            Map<String, Object> uiState,
            Instant now
    ) {
        mapper.patchTemplate(
                templateId,
                context.tenantId(),
                context.workspaceId(),
                writeJson(graph == null ? Map.of() : graph),
                writeJson(uiState == null ? Map.of() : uiState),
                now == null ? Instant.now() : now
        );
        return findByIdInScope(templateId, context);
    }

    /**
     * Publishes template, stores immutable version snapshot, and returns latest template.
     */
    @Override
    public WorkflowTemplate publish(String templateId, ExecutionContext context, Instant now) {
        WorkflowTemplateEntity current = mapper.selectByIdInScope(templateId, context.tenantId(), context.workspaceId());
        if (current == null) {
            return null;
        }

        Instant ts = now == null ? Instant.now() : now;
        int nextVersion = Math.max(current.currentVersion == null ? 0 : current.currentVersion, 0) + 1;
        String graphJson = defaultJson(current.graphJson);
        String schemaInputsJson = defaultJson(current.schemaInputsJson);
        String schemaOutputsJson = defaultJson(current.schemaOutputsJson);

        mapper.insertTemplateVersion(
                UUID.randomUUID().toString(),
                current.id,
                nextVersion,
                graphJson,
                schemaInputsJson,
                schemaOutputsJson,
                sha256Hex(graphJson),
                context.userId(),
                ts
        );

        mapper.publishTemplate(
                current.id,
                context.tenantId(),
                context.workspaceId(),
                nextVersion,
                ts
        );

        WorkflowTemplateEntity latest = mapper.selectByIdInScope(templateId, context.tenantId(), context.workspaceId());
        return latest == null ? null : toResource(latest);
    }

    /**
     * Returns true when user or roles have requested ACL permission on template.
     */
    @Override
    public boolean hasPermission(String templateId, ExecutionContext context, Permission permission, Instant now) {
        Instant ts = now == null ? Instant.now() : now;
        String permissionValue = permission == null ? Permission.READ.name() : permission.name();

        boolean userAllowed = mapper.hasPermissionForUser(
                context.tenantId(),
                context.workspaceId(),
                templateId,
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
                templateId,
                roles,
                permissionValue,
                ts
        );
    }

    private DataPermissionContext toDataPermissionContext(ExecutionContext context) {
        return new DataPermissionContext(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                context.roles(),
                context.policyVersion(),
                "workflow_template",
                Permission.READ.name()
        );
    }

    private WorkflowTemplate toResource(WorkflowTemplateEntity entity) {
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

        return new WorkflowTemplate(
                base,
                entity.name,
                entity.description,
                readJsonMap(entity.graphJson),
                readJsonMap(entity.schemaInputsJson),
                readJsonMap(entity.schemaOutputsJson),
                readJsonMap(entity.uiStateJson),
                entity.currentVersion == null ? 0 : entity.currentVersion
        );
    }

    private Visibility parseVisibility(String value) {
        if (value == null || value.isBlank()) {
            return Visibility.PRIVATE;
        }
        try {
            return Visibility.valueOf(value.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            return Visibility.PRIVATE;
        }
    }

    private String writeJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value == null ? Map.of() : value);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize workflow template json", ex);
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
            throw new IllegalStateException("failed to deserialize workflow template json", ex);
        }
    }

    private String defaultJson(String value) {
        return value == null || value.isBlank() ? "{}" : value;
    }

    private String sha256Hex(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(value.getBytes(java.nio.charset.StandardCharsets.UTF_8));
            StringBuilder builder = new StringBuilder(hash.length * 2);
            for (byte item : hash) {
                builder.append(String.format("%02x", item));
            }
            return builder.toString();
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException("SHA-256 not available", ex);
        }
    }
}
