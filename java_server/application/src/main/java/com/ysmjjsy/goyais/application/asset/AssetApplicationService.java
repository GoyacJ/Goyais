/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Asset application service for list/detail and domain-sugar command entrypoints.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.asset;

import com.ysmjjsy.goyais.application.command.CommandApplicationService;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.AclItem;
import com.ysmjjsy.goyais.contract.api.common.Asset;
import com.ysmjjsy.goyais.contract.api.common.AssetCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.AssetLineageResponse;
import com.ysmjjsy.goyais.contract.api.common.AssetUpdateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.CommandResource;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.ResourceBase;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Instant;
import java.time.format.DateTimeParseException;
import java.util.Base64;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import org.springframework.stereotype.Service;

/**
 * Coordinates asset read APIs and write APIs that map to canonical commands.
 */
@Service
public final class AssetApplicationService {
    private final CommandApplicationService commandService;
    private final AssetRepository assetRepository;

    /**
     * Creates asset application service with command and repository dependencies.
     * @param commandService TODO
     * @param assetRepository TODO
     */
    public AssetApplicationService(CommandApplicationService commandService, AssetRepository assetRepository) {
        this.commandService = commandService;
        this.assetRepository = assetRepository;
    }

    /**
     * Creates asset through command-first flow and returns accepted write response.
     * @param request TODO
     * @param file TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<Asset> create(AssetCreateRequest request, byte[] file, ExecutionContext context) {
        if (file == null || file.length == 0) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "file")
            );
        }

        String hash = sha256Hex(file);
        Map<String, Object> payload = new java.util.LinkedHashMap<>();
        if (request.name() != null) {
            payload.put("name", request.name());
        }
        if (request.type() != null) {
            payload.put("type", request.type());
        }
        if (request.mime() != null) {
            payload.put("mime", request.mime());
        }
        payload.put("hash", hash);
        payload.put("size", file.length);
        payload.put("visibility", request.visibility() == null ? Visibility.PRIVATE.name() : request.visibility().name());
        payload.put("fileBase64", Base64.getEncoder().encodeToString(file));

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("asset.upload", Map.copyOf(payload), request.visibility()),
                context
        );

        Asset asset = extractAsset(commandResponse.resource().result());
        return new WriteResponse<>(asset, commandResponse.commandRef());
    }

    /**
     * Updates one asset through command-first flow.
     * @param assetId TODO
     * @param request TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<Asset> update(String assetId, AssetUpdateRequest request, ExecutionContext context) {
        Map<String, Object> payload = new java.util.LinkedHashMap<>();
        payload.put("assetId", assetId);
        if (request != null && request.name() != null) {
            payload.put("name", request.name());
        }
        if (request != null && request.visibility() != null) {
            payload.put("visibility", request.visibility().name());
        }
        if (request != null && request.metadata() != null) {
            payload.put("metadata", request.metadata());
        }

        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("asset.update", Map.copyOf(payload), null),
                context
        );

        Asset asset = extractAsset(commandResponse.resource().result());
        return new WriteResponse<>(asset, commandResponse.commandRef());
    }

    /**
     * Deletes one asset through command-first flow.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    public WriteResponse<Asset> delete(String assetId, ExecutionContext context) {
        WriteResponse<CommandResource> commandResponse = commandService.create(
                new CommandCreateRequest("asset.delete", Map.of("assetId", assetId), null),
                context
        );

        Asset asset = extractAsset(commandResponse.resource().result());
        return new WriteResponse<>(asset, commandResponse.commandRef());
    }

    /**
     * Lists readable assets with deterministic pagination bounds.
     * @param context TODO
     * @param page TODO
     * @param pageSize TODO
     * @return TODO
     */
    public List<Asset> list(ExecutionContext context, int page, int pageSize) {
        return assetRepository.listReadable(context, normalizePage(page), normalizePageSize(pageSize));
    }

    /**
     * Returns count of readable assets for current context.
     * @param context TODO
     * @return TODO
     */
    public long count(ExecutionContext context) {
        return assetRepository.countReadable(context);
    }

    /**
     * Returns one readable asset by identifier.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    public Asset get(String assetId, ExecutionContext context) {
        return assetRepository.findReadableById(assetId, context);
    }

    /**
     * Returns lineage graph for one readable asset.
     * @param assetId TODO
     * @param context TODO
     * @return TODO
     */
    public AssetLineageResponse lineage(String assetId, ExecutionContext context) {
        return new AssetLineageResponse(assetId, assetRepository.listLineage(assetId, context));
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

    private Asset extractAsset(Map<String, Object> result) {
        if (result == null || !(result.get("asset") instanceof Map<?, ?> rawAsset)) {
            throw ContractException.of(500, "INTERNAL_ERROR", "error.internal");
        }
        Map<String, Object> assetMap = copyStringObjectMap(rawAsset);
        Map<String, Object> baseMap = copyStringObjectMap(assetMap.get("base"));

        ResourceBase base = new ResourceBase(
                requiredString(baseMap, "id"),
                requiredString(baseMap, "tenantId"),
                requiredString(baseMap, "workspaceId"),
                requiredString(baseMap, "ownerId"),
                parseVisibility(requiredString(baseMap, "visibility")),
                List.<AclItem>of(),
                requiredString(baseMap, "status"),
                parseInstant(baseMap.get("createdAt")),
                parseInstant(baseMap.get("updatedAt"))
        );

        return new Asset(
                base,
                requiredString(assetMap, "name"),
                requiredString(assetMap, "type"),
                requiredString(assetMap, "mime"),
                parseLong(assetMap.get("size")),
                requiredString(assetMap, "hash"),
                requiredString(assetMap, "uri"),
                copyStringObjectMap(assetMap.get("metadata"))
        );
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

    private Visibility parseVisibility(String raw) {
        try {
            return Visibility.valueOf(raw.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            return Visibility.PRIVATE;
        }
    }

    private Instant parseInstant(Object raw) {
        if (raw == null) {
            return Instant.now();
        }
        if (raw instanceof Instant instant) {
            return instant;
        }
        try {
            return Instant.parse(String.valueOf(raw));
        } catch (DateTimeParseException ex) {
            return Instant.now();
        }
    }

    private long parseLong(Object raw) {
        if (raw instanceof Number number) {
            return number.longValue();
        }
        if (raw == null) {
            return 0L;
        }
        try {
            return Long.parseLong(String.valueOf(raw));
        } catch (NumberFormatException ex) {
            return 0L;
        }
    }
}
