/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Internal mapper for workflow command result payload to contract records.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.workflow;

import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.AclItem;
import com.ysmjjsy.goyais.contract.api.common.ErrorBody;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.StepRun;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRun;
import com.ysmjjsy.goyais.contract.api.common.WorkflowRunEvent;
import com.ysmjjsy.goyais.contract.api.common.WorkflowTemplate;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.LinkedHashMap;
import java.util.Locale;
import java.util.Map;

final class WorkflowContractMapper {
    private WorkflowContractMapper() {
    }

    static WorkflowTemplate toWorkflowTemplate(Map<String, Object> payload) {
        Map<String, Object> source = copyObjectMap(payload);
        return new WorkflowTemplate(
                new ResourceBase(
                        requiredString(source, "id"),
                        requiredString(source, "tenantId"),
                        requiredString(source, "workspaceId"),
                        requiredString(source, "ownerId"),
                        parseVisibility(requiredString(source, "visibility")),
                        java.util.List.<AclItem>of(),
                        requiredString(source, "status"),
                        parseRequiredInstant(source.get("createdAt")),
                        parseRequiredInstant(source.get("updatedAt"))
                ),
                requiredString(source, "name"),
                readString(source, "description"),
                copyObjectMap(source.get("graph")),
                copyObjectMap(source.get("schemaInputs")),
                copyObjectMap(source.get("schemaOutputs")),
                copyObjectMap(source.get("uiState")),
                parseInt(source.get("currentVersion"))
        );
    }

    static WorkflowRun toWorkflowRun(Map<String, Object> payload) {
        Map<String, Object> source = copyObjectMap(payload);
        Instant startedAt = parseRequiredInstant(source.get("startedAt"));
        Instant finishedAt = parseNullableInstant(source.get("finishedAt"));

        Long durationMs = source.get("durationMs") == null ? null : parseLong(source.get("durationMs"));
        if (durationMs == null && finishedAt != null) {
            long computed = finishedAt.toEpochMilli() - startedAt.toEpochMilli();
            durationMs = Math.max(computed, 0L);
        }

        return new WorkflowRun(
                new ResourceBase(
                        requiredString(source, "id"),
                        requiredString(source, "tenantId"),
                        requiredString(source, "workspaceId"),
                        requiredString(source, "ownerId"),
                        parseVisibility(requiredString(source, "visibility")),
                        java.util.List.<AclItem>of(),
                        requiredString(source, "status"),
                        parseRequiredInstant(source.get("createdAt")),
                        parseRequiredInstant(source.get("updatedAt"))
                ),
                requiredString(source, "templateId"),
                parseInt(source.get("templateVersion")),
                parseInt(source.get("attempt")),
                readNullableString(source, "retryOfRunId"),
                readNullableString(source, "replayFromStepKey"),
                readNullableString(source, "traceId"),
                copyObjectMap(source.get("inputs")),
                copyObjectMap(source.get("outputs")),
                startedAt,
                finishedAt,
                durationMs,
                parseError(source.get("error"))
        );
    }

    static StepRun toStepRun(Map<String, Object> payload) {
        Map<String, Object> source = copyObjectMap(payload);
        Instant startedAt = parseRequiredInstant(source.get("startedAt"));
        Instant finishedAt = parseNullableInstant(source.get("finishedAt"));
        Long durationMs = source.get("durationMs") == null ? null : parseLong(source.get("durationMs"));

        return new StepRun(
                new ResourceBase(
                        requiredString(source, "id"),
                        requiredString(source, "tenantId"),
                        requiredString(source, "workspaceId"),
                        requiredString(source, "ownerId"),
                        parseVisibility(requiredString(source, "visibility")),
                        java.util.List.<AclItem>of(),
                        requiredString(source, "status"),
                        parseRequiredInstant(source.get("createdAt")),
                        parseRequiredInstant(source.get("updatedAt"))
                ),
                requiredString(source, "runId"),
                requiredString(source, "stepKey"),
                requiredString(source, "stepType"),
                parseInt(source.get("attempt")),
                readNullableString(source, "traceId"),
                copyObjectMap(source.get("input")),
                copyObjectMap(source.get("output")),
                copyObjectMap(source.get("artifacts")),
                readNullableString(source, "logRef"),
                startedAt,
                finishedAt,
                durationMs,
                parseError(source.get("error"))
        );
    }

    static WorkflowRunEvent toRunEvent(Map<String, Object> payload) {
        Map<String, Object> source = copyObjectMap(payload);
        return new WorkflowRunEvent(
                requiredString(source, "id"),
                requiredString(source, "runId"),
                requiredString(source, "tenantId"),
                requiredString(source, "workspaceId"),
                readNullableString(source, "stepKey"),
                requiredString(source, "eventType"),
                copyObjectMap(source.get("payload")),
                parseRequiredInstant(source.get("createdAt"))
        );
    }

    private static ErrorBody parseError(Object raw) {
        if (!(raw instanceof Map<?, ?> map)) {
            return null;
        }
        Map<String, Object> error = copyObjectMap(map);
        String code = readNullableString(error, "code");
        String messageKey = readNullableString(error, "messageKey");
        if ((code == null || code.isBlank()) && (messageKey == null || messageKey.isBlank())) {
            return null;
        }
        return new ErrorBody(code, messageKey, null);
    }

    private static Visibility parseVisibility(String raw) {
        try {
            return Visibility.valueOf(raw.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            return Visibility.PRIVATE;
        }
    }

    private static int parseInt(Object raw) {
        if (raw instanceof Number number) {
            return number.intValue();
        }
        if (raw == null) {
            return 0;
        }
        try {
            return Integer.parseInt(String.valueOf(raw));
        } catch (NumberFormatException ex) {
            return 0;
        }
    }

    private static Long parseLong(Object raw) {
        if (raw instanceof Number number) {
            return number.longValue();
        }
        if (raw == null) {
            return null;
        }
        try {
            return Long.parseLong(String.valueOf(raw));
        } catch (NumberFormatException ex) {
            return null;
        }
    }

    private static Instant parseRequiredInstant(Object raw) {
        Instant parsed = parseNullableInstant(raw);
        if (parsed == null) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return parsed;
    }

    private static Instant parseNullableInstant(Object raw) {
        if (raw == null) {
            return null;
        }
        if (raw instanceof Instant instant) {
            return instant;
        }
        String value = String.valueOf(raw).trim();
        if (value.isBlank()) {
            return null;
        }
        try {
            return Instant.parse(value);
        } catch (DateTimeParseException ex) {
            return null;
        }
    }

    private static String requiredString(Map<String, Object> payload, String key) {
        String value = readNullableString(payload, key);
        if (value == null || value.isBlank()) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return value;
    }

    private static String readString(Map<String, Object> payload, String key) {
        String value = readNullableString(payload, key);
        return value == null ? "" : value;
    }

    private static String readNullableString(Map<String, Object> payload, String key) {
        if (payload == null || payload.get(key) == null) {
            return null;
        }
        String value = String.valueOf(payload.get(key)).trim();
        return value.isBlank() ? null : value;
    }

    private static Map<String, Object> copyObjectMap(Object raw) {
        if (!(raw instanceof Map<?, ?> source)) {
            return Map.of();
        }
        Map<String, Object> copied = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            copied.put(String.valueOf(entry.getKey()), copyValue(entry.getValue()));
        }
        return Map.copyOf(copied);
    }

    private static Object copyValue(Object value) {
        if (value instanceof Map<?, ?> map) {
            return copyObjectMap(map);
        }
        if (value instanceof java.util.List<?> list) {
            java.util.ArrayList<Object> copied = new java.util.ArrayList<>(list.size());
            for (Object item : list) {
                copied.add(copyValue(item));
            }
            return copied;
        }
        return value;
    }
}
