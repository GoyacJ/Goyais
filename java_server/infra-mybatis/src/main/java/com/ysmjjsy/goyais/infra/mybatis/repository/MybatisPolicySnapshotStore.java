/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Durable policy snapshot store implementation backed by PostgreSQL.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.infra.mybatis.entity.PolicySnapshotEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.PolicySnapshotEntityMapper;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshot;
import com.ysmjjsy.goyais.kernel.security.PolicySnapshotStore;
import java.io.IOException;
import java.time.Instant;
import java.util.Set;
import org.springframework.stereotype.Repository;

/**
 * Reads and writes policy snapshots in the policies table.
 */
@Repository
public final class MybatisPolicySnapshotStore implements PolicySnapshotStore {
    private final PolicySnapshotEntityMapper mapper;
    private final ObjectMapper objectMapper;

    /**
     * Creates store with mapper and JSON codec dependencies.
     * @param mapper TODO
     * @param objectMapper TODO
     */
    public MybatisPolicySnapshotStore(PolicySnapshotEntityMapper mapper, ObjectMapper objectMapper) {
        this.mapper = mapper;
        this.objectMapper = objectMapper;
    }

    /**
     * Loads one policy snapshot by scope, or null when no row exists.
     * @param tenantId TODO
     * @param workspaceId TODO
     * @param userId TODO
     * @return TODO
     */
    @Override
    public PolicySnapshot load(String tenantId, String workspaceId, String userId) {
        PolicySnapshotEntity entity = mapper.selectByScope(tenantId, workspaceId, userId);
        if (entity == null) {
            return null;
        }

        return new PolicySnapshot(
                entity.getTenantId(),
                entity.getWorkspaceId(),
                entity.getUserId(),
                entity.getPolicyVersion(),
                readSet(entity.getRolesJson()),
                readSet(entity.getDeniedCommandTypesJson()),
                entity.getUpdatedAt()
        );
    }

    /**
     * Upserts one policy snapshot by scope key.
     * @param snapshot TODO
     */
    @Override
    public void upsert(PolicySnapshot snapshot) {
        PolicySnapshotEntity entity = new PolicySnapshotEntity();
        entity.setTenantId(snapshot.tenantId());
        entity.setWorkspaceId(snapshot.workspaceId());
        entity.setUserId(snapshot.userId());
        entity.setPolicyVersion(snapshot.policyVersion());
        entity.setRolesJson(writeSet(snapshot.roles()));
        entity.setDeniedCommandTypesJson(writeSet(snapshot.deniedCommandTypes()));
        entity.setUpdatedAt(snapshot.updatedAt() == null ? Instant.now() : snapshot.updatedAt());
        mapper.upsert(entity);
    }

    private String writeSet(Set<String> values) {
        try {
            return objectMapper.writeValueAsString(values == null ? Set.of() : values);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize policy set", ex);
        }
    }

    private Set<String> readSet(String payload) {
        if (payload == null || payload.isBlank()) {
            return Set.of();
        }
        try {
            return objectMapper.readValue(payload, new TypeReference<>() {
            });
        } catch (IOException ex) {
            throw new IllegalStateException("failed to deserialize policy set", ex);
        }
    }
}
