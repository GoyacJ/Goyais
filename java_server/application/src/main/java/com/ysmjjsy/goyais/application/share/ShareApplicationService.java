/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Share application service for domain-sugar APIs and read queries.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.share;

import com.ysmjjsy.goyais.application.command.CommandApplicationService;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Share;
import com.ysmjjsy.goyais.contract.api.common.ShareCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.ShareDeleteResult;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Set;
import org.springframework.stereotype.Service;

/**
 * Coordinates share read APIs and share write APIs that map to canonical commands.
 */
@Service
public final class ShareApplicationService {
    private final CommandApplicationService commandService;
    private final ShareRepository shareRepository;

    /**
     * Creates share application service with command and repository dependencies.
     * @param commandService TODO
     * @param shareRepository TODO
     */
    public ShareApplicationService(CommandApplicationService commandService, ShareRepository shareRepository) {
        this.commandService = commandService;
        this.shareRepository = shareRepository;
    }

    /**
     * Creates one share through command-first flow.
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<Share> create(ShareCreateRequest request, ExecutionContext context) {
        if (request == null) {
            throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
        }
        Map<String, Object> payload = new java.util.LinkedHashMap<>();
        if (request.resourceType() != null) {
            payload.put("resourceType", request.resourceType());
        }
        if (request.resourceId() != null) {
            payload.put("resourceId", request.resourceId());
        }
        if (request.subjectType() != null) {
            payload.put("subjectType", request.subjectType());
        }
        if (request.subjectId() != null) {
            payload.put("subjectId", request.subjectId());
        }
        if (request.permissions() != null) {
            payload.put("permissions", request.permissions());
        }
        if (request.expiresAt() != null) {
            payload.put("expiresAt", request.expiresAt().toString());
        }

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("share.create", Map.copyOf(payload), null),
                context
        );

        Share share = extractShare(commandResponse.resource().result());
        return new WriteResponse<>(share, commandResponse.commandRef());
    }

    /**
     * Deletes one share through command-first flow.
     * @param shareId TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<ShareDeleteResult> delete(String shareId, ExecutionContext context) {
        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("share.delete", Map.of("shareId", shareId), null),
                context
        );

        ShareDeleteResult result = extractDeleteResult(commandResponse.resource().result(), shareId);
        return new WriteResponse<>(result, commandResponse.commandRef());
    }

    /**
     * Lists shares with deterministic pagination bounds.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    public List<Share> list(ExecutionContext context, int page, int pageSize) {
        return shareRepository.list(context, normalizePage(page), normalizePageSize(pageSize));
    }

    /**
     * Returns share count in current tenant/workspace scope.
     * @param context TODO
     * @return TODO
     */
    public long count(ExecutionContext context) {
        return shareRepository.count(context);
    }

    private int normalizePage(int page) {
        return page <= 0 ? 1 : page;
    }

    private int normalizePageSize(int pageSize) {
        if (pageSize <= 0) {
            return 20;
        }
        return Math.min(pageSize, 200);
    }

    private Share extractShare(Map<String, Object> result) {
        if (result == null || !(result.get("share") instanceof Map<?, ?> rawShare)) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        Map<String, Object> share = copyStringObjectMap(rawShare);

        Set<Permission> permissions = parsePermissions(share.get("permissions"));
        return new Share(
                requiredString(share, "id"),
                requiredString(share, "tenantId"),
                requiredString(share, "workspaceId"),
                requiredString(share, "resourceType"),
                requiredString(share, "resourceId"),
                requiredString(share, "subjectType"),
                requiredString(share, "subjectId"),
                permissions,
                parseInstant(share.get("expiresAt")),
                requiredString(share, "createdBy"),
                parseRequiredInstant(share.get("createdAt"))
        );
    }

    private ShareDeleteResult extractDeleteResult(Map<String, Object> result, String shareId) {
        if (result != null && result.get("share") instanceof Map<?, ?> rawShare) {
            Map<String, Object> share = copyStringObjectMap(rawShare);
            return new ShareDeleteResult(
                    requiredString(share, "id"),
                    requiredString(share, "status")
            );
        }
        return new ShareDeleteResult(shareId, "deleted");
    }

    private Map<String, Object> copyStringObjectMap(Object raw) {
        if (!(raw instanceof Map<?, ?> source)) {
            return Map.of();
        }
        Map<String, Object> target = new java.util.LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            target.put(String.valueOf(entry.getKey()), entry.getValue());
        }
        return Map.copyOf(target);
    }

    private String requiredString(Map<String, Object> source, String key) {
        if (source == null || source.get(key) == null || String.valueOf(source.get(key)).isBlank()) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return String.valueOf(source.get(key)).trim();
    }

    private Set<Permission> parsePermissions(Object raw) {
        if (!(raw instanceof Iterable<?> iterable)) {
            return Set.of();
        }
        java.util.Set<Permission> values = new java.util.LinkedHashSet<>();
        for (Object item : iterable) {
            if (item == null) {
                continue;
            }
            try {
                values.add(Permission.valueOf(String.valueOf(item).trim().toUpperCase()));
            } catch (IllegalArgumentException ex) {
                // Ignore invalid values to keep response parsing resilient.
            }
        }
        Permission[] ordered = values.toArray(new Permission[0]);
        Arrays.sort(ordered);
        return Set.copyOf(Arrays.asList(ordered));
    }

    private Instant parseInstant(Object raw) {
        if (raw == null || String.valueOf(raw).isBlank()) {
            return null;
        }
        try {
            return Instant.parse(String.valueOf(raw));
        } catch (DateTimeParseException ex) {
            return null;
        }
    }

    private Instant parseRequiredInstant(Object raw) {
        Instant value = parseInstant(raw);
        if (value == null) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        return value;
    }
}
