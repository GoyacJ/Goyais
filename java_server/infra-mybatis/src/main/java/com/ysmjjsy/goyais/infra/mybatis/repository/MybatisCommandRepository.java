/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>MyBatisPlus implementation of command repository with SQL-layer data permission.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.command.CommandRepository;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.ErrorBody;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.infra.mybatis.entity.CommandEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.CommandEntityMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionResolver;
import java.io.IOException;
import java.util.List;
import java.util.Map;
import org.springframework.stereotype.Repository;

/**
 * Persists commands and applies data permission filters in SQL queries.
 */
@Repository
public final class MybatisCommandRepository implements CommandRepository {
    private final CommandEntityMapper mapper;
    private final DataPermissionResolver dataPermissionResolver;
    private final ObjectMapper objectMapper;

    /**
     * Creates repository with mapper, data-permission resolver, and JSON codec.
     * @param mapper TODO
     * @param dataPermissionResolver TODO
     * @param objectMapper TODO
     */
    public MybatisCommandRepository(
            CommandEntityMapper mapper,
            DataPermissionResolver dataPermissionResolver,
            ObjectMapper objectMapper
    ) {
        this.mapper = mapper;
        this.dataPermissionResolver = dataPermissionResolver;
        this.objectMapper = objectMapper;
    }

    /**
     * Persists one command resource into commands table.
     * @param resource TODO
     */
    @Override
    public void save(CommandResource resource) {
        CommandEntity entity = new CommandEntity();
        entity.setId(resource.base().id());
        entity.setTenantId(resource.base().tenantId());
        entity.setWorkspaceId(resource.base().workspaceId());
        entity.setOwnerId(resource.base().ownerId());
        entity.setVisibility(resource.base().visibility().name());
        entity.setStatus(resource.base().status());
        entity.setCommandType(resource.commandType());
        entity.setPayloadJson(writeJson(resource.payload()));
        entity.setTraceId(resource.traceId());
        entity.setResultJson(writeJson(resource.result()));
        entity.setAcceptedAt(resource.acceptedAt());
        entity.setCreatedAt(resource.base().createdAt());
        entity.setUpdatedAt(resource.base().updatedAt());

        if (resource.error() != null) {
            entity.setErrorCode(resource.error().code());
            entity.setErrorMessageKey(resource.error().messageKey());
        }

        mapper.insert(entity);
    }

    /**
     * Returns readable command resources ordered by created_at DESC and id DESC.
     * @param context TODO
     * @param limit TODO
     * @return TODO
     */
    @Override
    public List<CommandResource> listReadable(ExecutionContext context, int limit) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return mapper.selectReadableList(predicate, dp, limit).stream().map(this::toResource).toList();
    }

    /**
     * Returns one readable command resource by identifier, or null.
     * @param commandId TODO
     * @param context TODO
     * @return TODO
     */
    @Override
    public CommandResource findReadableById(String commandId, ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        CommandEntity entity = mapper.selectReadableById(commandId, predicate, dp);
        return entity == null ? null : toResource(entity);
    }

    private DataPermissionContext toDataPermissionContext(ExecutionContext context) {
        return new DataPermissionContext(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                context.roles(),
                context.policyVersion(),
                "command",
                "READ"
        );
    }

    private CommandResource toResource(CommandEntity entity) {
        ResourceBase base = new ResourceBase(
                entity.getId(),
                entity.getTenantId(),
                entity.getWorkspaceId(),
                entity.getOwnerId(),
                parseVisibility(entity.getVisibility()),
                List.of(),
                entity.getStatus(),
                entity.getCreatedAt(),
                entity.getUpdatedAt()
        );

        ErrorBody error = null;
        if (entity.getErrorCode() != null || entity.getErrorMessageKey() != null) {
            error = new ErrorBody(entity.getErrorCode(), entity.getErrorMessageKey(), null);
        }

        return new CommandResource(
                base,
                entity.getCommandType(),
                readJsonMap(entity.getPayloadJson()),
                entity.getAcceptedAt(),
                entity.getTraceId(),
                readJsonMap(entity.getResultJson()),
                error
        );
    }

    private Visibility parseVisibility(String value) {
        if (value == null || value.isBlank()) {
            return Visibility.PRIVATE;
        }
        try {
            return Visibility.valueOf(value);
        } catch (IllegalArgumentException ex) {
            return Visibility.PRIVATE;
        }
    }

    private String writeJson(Object value) {
        if (value == null) {
            return null;
        }
        try {
            return objectMapper.writeValueAsString(value);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize command payload", ex);
        }
    }

    private Map<String, Object> readJsonMap(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        try {
            return objectMapper.readValue(value, new TypeReference<>() {
            });
        } catch (IOException ex) {
            throw new IllegalStateException("failed to deserialize command payload", ex);
        }
    }
}
