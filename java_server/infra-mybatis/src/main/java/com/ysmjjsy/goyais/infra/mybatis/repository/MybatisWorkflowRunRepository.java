/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: MyBatisPlus implementation of workflow run repository and step/event read models.
 */
package com.ysmjjsy.goyais.infra.mybatis.repository;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ysmjjsy.goyais.application.workflow.WorkflowRunRepository;
import com.ysmjjsy.goyais.contract.api.common.AclItem;
import com.ysmjjsy.goyais.contract.api.common.ErrorBody;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import com.ysmjjsy.goyais.infra.mybatis.entity.StepRunEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowRunEntity;
import com.ysmjjsy.goyais.infra.mybatis.entity.WorkflowRunEventEntity;
import com.ysmjjsy.goyais.infra.mybatis.mapper.StepRunEntityMapper;
import com.ysmjjsy.goyais.infra.mybatis.mapper.WorkflowRunEntityMapper;
import com.ysmjjsy.goyais.infra.mybatis.mapper.WorkflowRunEventEntityMapper;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionContext;
import com.ysmjjsy.goyais.kernel.mybatis.DataPermissionResolver;
import java.io.IOException;
import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import org.springframework.stereotype.Repository;

/**
 * Persists workflow runs and exposes permission-aware run, step, and event queries.
 */
@Repository
public final class MybatisWorkflowRunRepository implements WorkflowRunRepository {
    private final WorkflowRunEntityMapper runMapper;
    private final StepRunEntityMapper stepMapper;
    private final WorkflowRunEventEntityMapper eventMapper;
    private final DataPermissionResolver dataPermissionResolver;
    private final ObjectMapper objectMapper;

    /**
     * Creates repository with run/step/event mappers, permission resolver, and JSON codec.
     */
    public MybatisWorkflowRunRepository(
            WorkflowRunEntityMapper runMapper,
            StepRunEntityMapper stepMapper,
            WorkflowRunEventEntityMapper eventMapper,
            DataPermissionResolver dataPermissionResolver,
            ObjectMapper objectMapper
    ) {
        this.runMapper = runMapper;
        this.stepMapper = stepMapper;
        this.eventMapper = eventMapper;
        this.dataPermissionResolver = dataPermissionResolver;
        this.objectMapper = objectMapper;
    }

    /**
     * Creates one run row, one initial step row, and bootstrap run events.
     */
    @Override
    public WorkflowRun createRun(
            ExecutionContext context,
            WorkflowTemplate template,
            Visibility visibility,
            String mode,
            String fromStepKey,
            boolean testNode,
            Map<String, Object> inputs,
            Instant now
    ) {
        Instant ts = now == null ? Instant.now() : now;
        String normalizedMode = mode == null ? "sync" : mode.trim().toLowerCase(Locale.ROOT);
        String runStatus = toRunStatus(normalizedMode);
        String stepStatus = toStepStatus(normalizedMode);

        WorkflowRunEntity run = new WorkflowRunEntity();
        run.id = UUID.randomUUID().toString();
        run.tenantId = context.tenantId();
        run.workspaceId = context.workspaceId();
        run.ownerId = context.userId();
        run.traceId = context.traceId();
        run.visibility = (visibility == null ? Visibility.PRIVATE : visibility).name();
        run.aclJson = "[]";
        run.templateId = template.base().id();
        run.templateVersion = Math.max(template.currentVersion(), 1);
        run.attempt = 1;
        run.retryOfRunId = null;
        run.replayFromStepKey = emptyToNull(fromStepKey);
        run.commandId = null;
        run.inputsJson = writeJson(inputs == null ? Map.of() : inputs);
        run.outputsJson = "succeeded".equals(runStatus)
                ? writeJson(Map.of("status", "succeeded"))
                : writeJson(Map.of());
        run.status = runStatus;
        run.errorCode = "failed".equals(runStatus) ? "WORKFLOW_RUN_FAILED" : null;
        run.messageKey = "failed".equals(runStatus) ? "error.workflow.run_failed" : null;
        run.startedAt = ts;
        run.finishedAt = "running".equals(runStatus) ? null : ts;
        run.createdAt = ts;
        run.updatedAt = ts;
        runMapper.insert(run);

        String stepKey = emptyToDefault(fromStepKey, "start");
        StepRunEntity step = new StepRunEntity();
        step.id = UUID.randomUUID().toString();
        step.runId = run.id;
        step.tenantId = context.tenantId();
        step.workspaceId = context.workspaceId();
        step.ownerId = context.userId();
        step.traceId = context.traceId();
        step.visibility = run.visibility;
        step.stepKey = stepKey;
        step.stepType = testNode ? "test" : "task";
        step.attempt = 1;
        step.inputJson = writeJson(inputs == null ? Map.of() : inputs);
        step.outputJson = "succeeded".equals(stepStatus)
                ? writeJson(Map.of("status", "succeeded"))
                : writeJson(Map.of());
        step.artifactsJson = writeJson(Map.of());
        step.logRef = null;
        step.status = stepStatus;
        step.errorCode = "failed".equals(stepStatus) ? "WORKFLOW_STEP_FAILED" : null;
        step.messageKey = "failed".equals(stepStatus) ? "error.workflow.step_failed" : null;
        step.startedAt = ts;
        step.finishedAt = "running".equals(stepStatus) ? null : ts;
        step.createdAt = ts;
        step.updatedAt = ts;
        stepMapper.insert(step);

        appendRunEvent(run.id, run.tenantId, run.workspaceId, null, "workflow.run.started", Map.of(
                "status", run.status
        ), ts);
        appendRunEvent(run.id, run.tenantId, run.workspaceId, step.stepKey, "workflow.step.started", Map.of(
                "stepKey", step.stepKey,
                "stepType", step.stepType,
                "status", step.status
        ), ts.plusNanos(1_000));

        if ("succeeded".equals(step.status)) {
            appendRunEvent(run.id, run.tenantId, run.workspaceId, step.stepKey, "workflow.step.succeeded", Map.of(
                    "stepKey", step.stepKey,
                    "status", "succeeded"
            ), ts.plusNanos(2_000));
            appendRunEvent(run.id, run.tenantId, run.workspaceId, null, "workflow.run.succeeded", Map.of(
                    "status", "succeeded"
            ), ts.plusNanos(3_000));
        } else if ("failed".equals(step.status)) {
            appendRunEvent(run.id, run.tenantId, run.workspaceId, step.stepKey, "workflow.step.failed", Map.of(
                    "stepKey", step.stepKey,
                    "status", "failed",
                    "code", "WORKFLOW_STEP_FAILED"
            ), ts.plusNanos(2_000));
            appendRunEvent(run.id, run.tenantId, run.workspaceId, null, "workflow.run.failed", Map.of(
                    "status", "failed",
                    "code", "WORKFLOW_RUN_FAILED"
            ), ts.plusNanos(3_000));
        }

        WorkflowRunEntity latest = runMapper.selectByIdInScope(run.id, context.tenantId(), context.workspaceId());
        return latest == null ? null : toRun(latest);
    }

    /**
     * Returns one run in tenant/workspace scope without ACL read filtering.
     */
    @Override
    public WorkflowRun findByIdInScope(String runId, ExecutionContext context) {
        WorkflowRunEntity entity = runMapper.selectByIdInScope(runId, context.tenantId(), context.workspaceId());
        return entity == null ? null : toRun(entity);
    }

    /**
     * Returns one readable run by id, or null when inaccessible.
     */
    @Override
    public WorkflowRun findReadableById(String runId, ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        WorkflowRunEntity entity = runMapper.selectReadableById(runId, predicate, dp);
        return entity == null ? null : toRun(entity);
    }

    /**
     * Returns readable run list with deterministic descending order.
     */
    @Override
    public List<WorkflowRun> listReadable(ExecutionContext context, int page, int pageSize) {
        int normalizedPage = Math.max(page, 1);
        int normalizedPageSize = pageSize <= 0 ? 20 : Math.min(pageSize, 200);
        int offset = (normalizedPage - 1) * normalizedPageSize;

        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return runMapper.selectReadableList(predicate, dp, normalizedPageSize, offset).stream()
                .map(this::toRun)
                .toList();
    }

    /**
     * Returns count of readable runs for current context.
     */
    @Override
    public long countReadable(ExecutionContext context) {
        DataPermissionContext dp = toDataPermissionContext(context);
        String predicate = dataPermissionResolver.resolveReadPredicate("c", dp);
        return runMapper.countReadable(predicate, dp);
    }

    /**
     * Returns true when user or roles have requested ACL permission on run.
     */
    @Override
    public boolean hasPermission(String runId, ExecutionContext context, Permission permission, Instant now) {
        Instant ts = now == null ? Instant.now() : now;
        String permissionValue = permission == null ? Permission.READ.name() : permission.name();

        boolean userAllowed = runMapper.hasPermissionForUser(
                context.tenantId(),
                context.workspaceId(),
                runId,
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

        return runMapper.hasPermissionForRoles(
                context.tenantId(),
                context.workspaceId(),
                runId,
                roles,
                permissionValue,
                ts
        );
    }

    /**
     * Cancels one active run and active steps, then emits cancellation events.
     */
    @Override
    public WorkflowRun cancelRun(String runId, ExecutionContext context, Instant now) {
        Instant ts = now == null ? Instant.now() : now;
        List<String> activeStepKeys = stepMapper.selectActiveStepKeys(
                context.tenantId(),
                context.workspaceId(),
                runId
        );

        int runAffected = runMapper.cancelActiveRun(
                runId,
                context.tenantId(),
                context.workspaceId(),
                ts,
                ts
        );
        if (runAffected > 0) {
            stepMapper.cancelActiveSteps(
                    context.tenantId(),
                    context.workspaceId(),
                    runId,
                    ts,
                    ts
            );
        }

        WorkflowRunEntity latest = runMapper.selectByIdInScope(runId, context.tenantId(), context.workspaceId());
        if (latest == null) {
            return null;
        }

        if (runAffected > 0) {
            appendRunEvent(latest.id, latest.tenantId, latest.workspaceId, null, "workflow.run.canceled", Map.of(
                    "status", "canceled"
            ), ts);
            for (int index = 0; index < activeStepKeys.size(); index += 1) {
                String stepKey = activeStepKeys.get(index);
                appendRunEvent(latest.id, latest.tenantId, latest.workspaceId, stepKey, "workflow.step.canceled", Map.of(
                        "stepKey", stepKey,
                        "status", "canceled"
                ), ts.plusNanos((index + 1L) * 1_000L));
            }
        }
        return toRun(latest);
    }

    /**
     * Returns step runs for one readable run with deterministic descending order.
     */
    @Override
    public List<StepRun> listSteps(String runId, ExecutionContext context, int page, int pageSize) {
        if (findReadableById(runId, context) == null) {
            return List.of();
        }

        int normalizedPage = Math.max(page, 1);
        int normalizedPageSize = pageSize <= 0 ? 20 : Math.min(pageSize, 200);
        int offset = (normalizedPage - 1) * normalizedPageSize;
        return stepMapper.selectByRun(context.tenantId(), context.workspaceId(), runId, normalizedPageSize, offset)
                .stream()
                .map(this::toStepRun)
                .toList();
    }

    /**
     * Returns count of step runs for one readable run.
     */
    @Override
    public long countSteps(String runId, ExecutionContext context) {
        if (findReadableById(runId, context) == null) {
            return 0;
        }
        return stepMapper.countByRun(context.tenantId(), context.workspaceId(), runId);
    }

    /**
     * Returns run events for one readable run ordered by created_at ASC and id ASC.
     */
    @Override
    public List<WorkflowRunEvent> listEvents(String runId, ExecutionContext context) {
        if (findReadableById(runId, context) == null) {
            return List.of();
        }
        return eventMapper.selectByRun(context.tenantId(), context.workspaceId(), runId).stream()
                .map(this::toRunEvent)
                .toList();
    }

    private DataPermissionContext toDataPermissionContext(ExecutionContext context) {
        return new DataPermissionContext(
                context.tenantId(),
                context.workspaceId(),
                context.userId(),
                context.roles(),
                context.policyVersion(),
                "workflow_run",
                Permission.READ.name()
        );
    }

    private String toRunStatus(String mode) {
        return switch (mode) {
            case "running" -> "running";
            case "fail" -> "failed";
            default -> "succeeded";
        };
    }

    private String toStepStatus(String mode) {
        return switch (mode) {
            case "running" -> "running";
            case "fail" -> "failed";
            default -> "succeeded";
        };
    }

    private WorkflowRun toRun(WorkflowRunEntity entity) {
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

        Long durationMs = null;
        if (entity.startedAt != null && entity.finishedAt != null) {
            durationMs = Math.max(entity.finishedAt.toEpochMilli() - entity.startedAt.toEpochMilli(), 0L);
        }
        ErrorBody error = null;
        if ((entity.errorCode != null && !entity.errorCode.isBlank())
                || (entity.messageKey != null && !entity.messageKey.isBlank())) {
            error = new ErrorBody(entity.errorCode, entity.messageKey, null);
        }

        return new WorkflowRun(
                base,
                entity.templateId,
                entity.templateVersion == null ? 0 : entity.templateVersion,
                entity.attempt == null ? 1 : entity.attempt,
                entity.retryOfRunId,
                entity.replayFromStepKey,
                entity.traceId,
                readJsonMap(entity.inputsJson),
                readJsonMap(entity.outputsJson),
                entity.startedAt,
                entity.finishedAt,
                durationMs,
                error
        );
    }

    private StepRun toStepRun(StepRunEntity entity) {
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

        Long durationMs = null;
        if (entity.startedAt != null && entity.finishedAt != null) {
            durationMs = Math.max(entity.finishedAt.toEpochMilli() - entity.startedAt.toEpochMilli(), 0L);
        }
        ErrorBody error = null;
        if ((entity.errorCode != null && !entity.errorCode.isBlank())
                || (entity.messageKey != null && !entity.messageKey.isBlank())) {
            error = new ErrorBody(entity.errorCode, entity.messageKey, null);
        }

        return new StepRun(
                base,
                entity.runId,
                entity.stepKey,
                entity.stepType,
                entity.attempt == null ? 1 : entity.attempt,
                entity.traceId,
                readJsonMap(entity.inputJson),
                readJsonMap(entity.outputJson),
                readJsonMap(entity.artifactsJson),
                entity.logRef,
                entity.startedAt,
                entity.finishedAt,
                durationMs,
                error
        );
    }

    private WorkflowRunEvent toRunEvent(WorkflowRunEventEntity entity) {
        return new WorkflowRunEvent(
                entity.id,
                entity.runId,
                entity.tenantId,
                entity.workspaceId,
                entity.stepKey,
                entity.eventType,
                readJsonMap(entity.payloadJson),
                entity.createdAt
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

    private void appendRunEvent(
            String runId,
            String tenantId,
            String workspaceId,
            String stepKey,
            String eventType,
            Map<String, Object> payload,
            Instant createdAt
    ) {
        WorkflowRunEventEntity entity = new WorkflowRunEventEntity();
        entity.id = UUID.randomUUID().toString();
        entity.runId = runId;
        entity.tenantId = tenantId;
        entity.workspaceId = workspaceId;
        entity.stepKey = stepKey;
        entity.eventType = eventType;
        entity.payloadJson = writeJson(payload == null ? Map.of() : payload);
        entity.createdAt = createdAt == null ? Instant.now() : createdAt;
        eventMapper.insert(entity);
    }

    private Map<String, Object> readJsonMap(String value) {
        if (value == null || value.isBlank()) {
            return Map.of();
        }
        try {
            return objectMapper.readValue(value, new TypeReference<>() {
            });
        } catch (IOException ex) {
            throw new IllegalStateException("failed to deserialize workflow run json", ex);
        }
    }

    private String writeJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value == null ? Map.of() : value);
        } catch (JsonProcessingException ex) {
            throw new IllegalStateException("failed to serialize workflow run json", ex);
        }
    }

    private String emptyToDefault(String value, String fallback) {
        return value == null || value.isBlank() ? fallback : value.trim();
    }

    private String emptyToNull(String value) {
        if (value == null || value.isBlank()) {
            return null;
        }
        return value.trim();
    }
}
