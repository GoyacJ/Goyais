/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command handler for share create/delete actions.
 */
package com.ysmjjsy.goyais.application.share;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Share;
import com.ysmjjsy.goyais.contract.api.common.ShareCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

/**
 * Executes share commands with owner/ACL.SHARE authorization checks.
 */
@Component
@Order(220)
public final class ShareCommandHandler implements CommandHandler {
    private final ShareRepository shareRepository;
    private final boolean aclRoleSubjectEnabled;

    /**
     * Creates handler with share repository and role-subject feature switch.
     */
    public ShareCommandHandler(
            ShareRepository shareRepository,
            @Value("${goyais.feature.acl-role-subject-enabled:true}") boolean aclRoleSubjectEnabled
    ) {
        this.shareRepository = shareRepository;
        this.aclRoleSubjectEnabled = aclRoleSubjectEnabled;
    }

    /**
     * Returns true when command type belongs to share domain actions.
     */
    @Override
    public boolean supports(String commandType) {
        return "share.create".equals(commandType) || "share.delete".equals(commandType);
    }

    /**
     * Executes share command and returns API-compatible result payload.
     */
    @Override
    public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
        return switch (request.commandType()) {
            case "share.create" -> handleCreate(request.payload(), context);
            case "share.delete" -> handleDelete(request.payload(), context);
            default -> throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
        };
    }

    private Map<String, Object> handleCreate(Map<String, Object> payload, ExecutionContext context) {
        String resourceType = normalizeResourceType(requiredString(payload, "resourceType"));
        String resourceId = requiredString(payload, "resourceId");
        String subjectType = normalizeSubjectType(requiredString(payload, "subjectType"));
        String subjectId = requiredString(payload, "subjectId");
        if ("role".equals(subjectType) && !aclRoleSubjectEnabled) {
            throw ContractException.of(
                    400,
                    "INVALID_SHARE_REQUEST",
                    "error.share.invalid_request",
                    Map.of("field", "subjectType")
            );
        }

        Set<Permission> permissions = parsePermissions(payload.get("permissions"));
        Instant expiresAt = parseExpiresAt(payload.get("expiresAt"));

        ShareResourceScope scope = shareRepository.findResourceScope(resourceType, resourceId, context);
        if (scope == null) {
            throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
        }

        boolean allowed = context.userId().equals(scope.ownerId())
                || shareRepository.hasResourcePermission(resourceType, resourceId, context, Permission.SHARE, Instant.now());
        if (!allowed) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "permission_denied")
            );
        }

        Share created = shareRepository.create(
                new ShareCreateRequest(resourceType, resourceId, subjectType, subjectId, permissions, expiresAt),
                context,
                Instant.now()
        );
        return Map.of("share", toSharePayload(created));
    }

    private Map<String, Object> handleDelete(Map<String, Object> payload, ExecutionContext context) {
        String shareId = requiredString(payload, "shareId");
        boolean deleted = shareRepository.deleteByIdAndCreator(shareId, context);
        if (!deleted) {
            throw ContractException.of(404, "SHARE_NOT_FOUND", "error.share.not_found");
        }

        return Map.of(
                "share",
                Map.of(
                        "id", shareId,
                        "status", "deleted"
                )
        );
    }

    private String requiredString(Map<String, Object> payload, String key) {
        if (payload == null || payload.get(key) == null || String.valueOf(payload.get(key)).isBlank()) {
            throw ContractException.of(
                    400,
                    "INVALID_SHARE_REQUEST",
                    "error.share.invalid_request",
                    Map.of("field", key)
            );
        }
        return String.valueOf(payload.get(key)).trim();
    }

    private String normalizeResourceType(String value) {
        String normalized = value.toLowerCase();
        if (!"command".equals(normalized) && !"asset".equals(normalized)) {
            throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
        }
        return normalized;
    }

    private String normalizeSubjectType(String value) {
        String normalized = value.toLowerCase();
        if (!"user".equals(normalized) && !"role".equals(normalized)) {
            throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
        }
        return normalized;
    }

    private Set<Permission> parsePermissions(Object raw) {
        if (!(raw instanceof Iterable<?> iterable)) {
            throw ContractException.of(
                    400,
                    "INVALID_SHARE_REQUEST",
                    "error.share.invalid_request",
                    Map.of("field", "permissions")
            );
        }

        Set<Permission> permissions = new LinkedHashSet<>();
        for (Object item : iterable) {
            if (item == null || String.valueOf(item).isBlank()) {
                throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
            }
            try {
                permissions.add(Permission.valueOf(String.valueOf(item).trim().toUpperCase()));
            } catch (IllegalArgumentException ex) {
                throw ContractException.of(400, "INVALID_SHARE_REQUEST", "error.share.invalid_request");
            }
        }

        if (permissions.isEmpty()) {
            throw ContractException.of(
                    400,
                    "INVALID_SHARE_REQUEST",
                    "error.share.invalid_request",
                    Map.of("field", "permissions")
            );
        }

        Permission[] ordered = permissions.toArray(new Permission[0]);
        Arrays.sort(ordered);
        return Set.copyOf(Arrays.asList(ordered));
    }

    private Instant parseExpiresAt(Object raw) {
        if (raw == null || String.valueOf(raw).isBlank()) {
            return null;
        }
        try {
            return Instant.parse(String.valueOf(raw).trim());
        } catch (DateTimeParseException ex) {
            throw ContractException.of(
                    400,
                    "INVALID_SHARE_REQUEST",
                    "error.share.invalid_request",
                    Map.of("field", "expiresAt")
            );
        }
    }

    private Map<String, Object> toSharePayload(Share share) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("id", share.id());
        payload.put("tenantId", share.tenantId());
        payload.put("workspaceId", share.workspaceId());
        payload.put("resourceType", share.resourceType());
        payload.put("resourceId", share.resourceId());
        payload.put("subjectType", share.subjectType());
        payload.put("subjectId", share.subjectId());
        payload.put("permissions", share.permissions());
        payload.put("expiresAt", share.expiresAt());
        payload.put("createdBy", share.createdBy());
        payload.put("createdAt", share.createdAt());
        return payload;
    }
}
