/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Command handler for asset upload/update/delete domain actions.
 */
package com.ysmjjsy.goyais.application.asset;

import com.ysmjjsy.goyais.application.command.CommandHandler;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.capability.storage.ObjectStorage;
import com.ysmjjsy.goyais.contract.api.common.Asset;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.io.ByteArrayInputStream;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.annotation.Order;
import org.springframework.stereotype.Component;

/**
 * Executes asset command payloads and emits API-compatible asset result maps.
 */
@Component
@Order(200)
public final class AssetCommandHandler implements CommandHandler {
    private final AssetRepository assetRepository;
    private final ObjectStorage objectStorage;
    private final String bucket;
    private final boolean lifecycleEnabled;

    /**
     * Creates handler with repository, object storage, and feature switches.
     */
    public AssetCommandHandler(
            AssetRepository assetRepository,
            ObjectStorage objectStorage,
            @Value("${goyais.storage.bucket:assets}") String bucket,
            @Value("${goyais.feature.asset-lifecycle-enabled:true}") boolean lifecycleEnabled
    ) {
        this.assetRepository = assetRepository;
        this.objectStorage = objectStorage;
        this.bucket = bucket;
        this.lifecycleEnabled = lifecycleEnabled;
    }

    /**
     * Returns true when command type belongs to asset domain actions.
     */
    @Override
    public boolean supports(String commandType) {
        return "asset.upload".equals(commandType)
                || "asset.update".equals(commandType)
                || "asset.delete".equals(commandType);
    }

    /**
     * Executes asset command and returns command result payload.
     */
    @Override
    public Map<String, Object> execute(CommandCreateRequest request, ExecutionContext context) {
        return switch (request.commandType()) {
            case "asset.upload" -> handleUpload(request, context);
            case "asset.update" -> {
                ensureLifecycleEnabled();
                yield handleUpdate(request, context);
            }
            case "asset.delete" -> {
                ensureLifecycleEnabled();
                yield handleDelete(request, context);
            }
            default -> throw ContractException.of(400, "INVALID_ASSET_REQUEST", "error.asset.invalid_request");
        };
    }

    private Map<String, Object> handleUpload(CommandCreateRequest request, ExecutionContext context) {
        Map<String, Object> payload = request.payload();
        String fileBase64 = requiredString(payload, "fileBase64");

        byte[] content;
        try {
            content = Base64.getDecoder().decode(fileBase64);
        } catch (IllegalArgumentException ex) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "fileBase64")
            );
        }
        if (content.length == 0) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "file")
            );
        }

        String hash = optionalString(payload, "hash");
        String computedHash = sha256Hex(content);
        if (hash == null || hash.isBlank()) {
            hash = computedHash;
        } else if (!computedHash.equalsIgnoreCase(hash.trim())) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "hash", "reason", "hash_mismatch")
            );
        }

        String name = defaultIfBlank(optionalString(payload, "name"), "asset-" + UUID.randomUUID());
        String type = defaultIfBlank(optionalString(payload, "type"), "file");
        String mime = defaultIfBlank(optionalString(payload, "mime"), "application/octet-stream");
        Visibility visibility = normalizeCreateVisibility(payload, request.visibility());

        String assetId = UUID.randomUUID().toString();
        String objectKey = context.tenantId() + "/" + context.workspaceId() + "/" + assetId;
        objectStorage.put(bucket, objectKey, new ByteArrayInputStream(content), mime);
        String uri = "storage://" + bucket + "/" + objectKey;

        Asset asset = assetRepository.create(
                context,
                name,
                type,
                mime,
                content.length,
                hash,
                uri,
                visibility,
                Map.of(),
                Instant.now()
        );

        return Map.of("asset", toAssetPayload(asset));
    }

    private Map<String, Object> handleUpdate(CommandCreateRequest request, ExecutionContext context) {
        Map<String, Object> payload = request.payload();
        String assetId = requiredString(payload, "assetId");

        Asset current = assetRepository.findByIdInScope(assetId, context);
        if (current == null || "deleted".equalsIgnoreCase(current.base().status())) {
            throw ContractException.of(404, "ASSET_NOT_FOUND", "error.asset.not_found");
        }
        ensureWritePermission(current, context);

        String name = optionalString(payload, "name");
        Visibility visibility = null;
        if (payload != null && payload.get("visibility") != null) {
            visibility = parseVisibility(String.valueOf(payload.get("visibility")));
            if (visibility == Visibility.TENANT || visibility == Visibility.PUBLIC) {
                throw ContractException.of(
                        403,
                        "FORBIDDEN",
                        "error.authz.forbidden",
                        Map.of("reason", "visibility_escalation_not_allowed")
                );
            }
        }

        boolean metadataProvided = payload != null && payload.containsKey("metadata");
        Map<String, Object> metadata = null;
        if (metadataProvided) {
            Object raw = payload.get("metadata");
            if (raw == null) {
                metadata = Map.of();
            } else if (raw instanceof Map<?, ?> mapValue) {
                metadata = copyStringObjectMap(mapValue);
            } else {
                throw ContractException.of(
                        400,
                        "INVALID_ASSET_REQUEST",
                        "error.asset.invalid_request",
                        Map.of("field", "metadata")
                );
            }
        }

        if (isBlank(name) && visibility == null && !metadataProvided) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("reason", "empty_patch")
            );
        }

        Asset updated = assetRepository.update(
                assetId,
                context,
                isBlank(name) ? null : name.trim(),
                visibility,
                metadata,
                metadataProvided,
                Instant.now()
        );
        return Map.of("asset", toAssetPayload(updated));
    }

    private Map<String, Object> handleDelete(CommandCreateRequest request, ExecutionContext context) {
        Map<String, Object> payload = request.payload();
        String assetId = requiredString(payload, "assetId");

        Asset current = assetRepository.findByIdInScope(assetId, context);
        if (current == null || "deleted".equalsIgnoreCase(current.base().status())) {
            throw ContractException.of(404, "ASSET_NOT_FOUND", "error.asset.not_found");
        }
        ensureWritePermission(current, context);

        deleteObjectQuietly(current.uri());
        Asset deleted = assetRepository.markDeleted(assetId, context, Instant.now());
        return Map.of("asset", toAssetPayload(deleted));
    }

    private void ensureLifecycleEnabled() {
        if (!lifecycleEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.asset.not_implemented");
        }
    }

    private void ensureWritePermission(Asset asset, ExecutionContext context) {
        if (asset.base().ownerId().equals(context.userId())) {
            return;
        }
        boolean hasPermission = assetRepository.hasPermission(asset.base().id(), context, Permission.WRITE, Instant.now());
        if (!hasPermission) {
            throw ContractException.of(
                    403,
                    "FORBIDDEN",
                    "error.authz.forbidden",
                    Map.of("reason", "permission_denied")
            );
        }
    }

    private void deleteObjectQuietly(String uri) {
        if (isBlank(uri) || !uri.startsWith("storage://")) {
            return;
        }
        String raw = uri.substring("storage://".length());
        int slash = raw.indexOf('/');
        if (slash <= 0 || slash + 1 >= raw.length()) {
            return;
        }
        String objectBucket = raw.substring(0, slash);
        String key = raw.substring(slash + 1);
        objectStorage.delete(objectBucket, key);
    }

    private Visibility normalizeCreateVisibility(Map<String, Object> payload, Visibility fallback) {
        if (payload != null && payload.get("visibility") != null) {
            return parseVisibility(String.valueOf(payload.get("visibility")));
        }
        return fallback == null ? Visibility.PRIVATE : fallback;
    }

    private Visibility parseVisibility(String raw) {
        if (raw == null || raw.isBlank()) {
            return Visibility.PRIVATE;
        }
        try {
            return Visibility.valueOf(raw.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "visibility")
            );
        }
    }

    private String requiredString(Map<String, Object> payload, String key) {
        String value = optionalString(payload, key);
        if (value == null || value.isBlank()) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", key)
            );
        }
        return value.trim();
    }

    private String optionalString(Map<String, Object> payload, String key) {
        if (payload == null || payload.get(key) == null) {
            return null;
        }
        return String.valueOf(payload.get(key));
    }

    private String defaultIfBlank(String value, String fallback) {
        return isBlank(value) ? fallback : value.trim();
    }

    private boolean isBlank(String value) {
        return value == null || value.isBlank();
    }

    private String sha256Hex(byte[] bytes) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(bytes);
            StringBuilder builder = new StringBuilder(hash.length * 2);
            for (byte item : hash) {
                builder.append(String.format("%02x", item));
            }
            return builder.toString();
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException("SHA-256 not available", ex);
        }
    }

    private Map<String, Object> toAssetPayload(Asset asset) {
        Map<String, Object> base = new LinkedHashMap<>();
        base.put("id", asset.base().id());
        base.put("tenantId", asset.base().tenantId());
        base.put("workspaceId", asset.base().workspaceId());
        base.put("ownerId", asset.base().ownerId());
        base.put("visibility", asset.base().visibility().name());
        base.put("acl", asset.base().acl());
        base.put("status", asset.base().status());
        base.put("createdAt", asset.base().createdAt());
        base.put("updatedAt", asset.base().updatedAt());

        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("base", base);
        payload.put("name", asset.name());
        payload.put("type", asset.type());
        payload.put("mime", asset.mime());
        payload.put("size", asset.size());
        payload.put("hash", asset.hash());
        payload.put("uri", asset.uri());
        payload.put("metadata", asset.metadata() == null ? Map.of() : asset.metadata());
        return payload;
    }

    private Map<String, Object> copyStringObjectMap(Map<?, ?> source) {
        Map<String, Object> target = new LinkedHashMap<>();
        for (Map.Entry<?, ?> entry : source.entrySet()) {
            if (entry.getKey() == null) {
                continue;
            }
            target.put(String.valueOf(entry.getKey()), entry.getValue());
        }
        return Map.copyOf(target);
    }
}
