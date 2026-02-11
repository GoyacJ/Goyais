/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Asset REST controller aligned with Go-compatible /api/v1/assets routes.
 */
package com.ysmjjsy.goyais.adapter.rest;

import com.ysmjjsy.goyais.application.asset.AssetApplicationService;
import com.ysmjjsy.goyais.application.common.ContractException;
import com.ysmjjsy.goyais.contract.api.common.Asset;
import com.ysmjjsy.goyais.contract.api.common.AssetCreateRequest;
import com.ysmjjsy.goyais.contract.api.common.AssetLineageResponse;
import com.ysmjjsy.goyais.contract.api.common.AssetUpdateRequest;
import com.ysmjjsy.goyais.contract.api.common.ErrorEnvelope;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.contract.api.common.WriteResponse;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import jakarta.servlet.http.HttpServletRequest;
import java.io.IOException;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.Authentication;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

/**
 * Exposes asset list/detail and domain-sugar write APIs.
 */
@RestController
@RequestMapping("/api/v1/assets")
public final class AssetController {
    private final AssetApplicationService assetService;
    private final RequestExecutionContextFactory executionContextFactory;
    private final boolean assetLifecycleEnabled;

    /**
     * Creates controller with asset service and execution-context resolver.
     */
    public AssetController(
            AssetApplicationService assetService,
            RequestExecutionContextFactory executionContextFactory,
            @Value("${goyais.feature.asset-lifecycle-enabled:true}") boolean assetLifecycleEnabled
    ) {
        this.assetService = assetService;
        this.executionContextFactory = executionContextFactory;
        this.assetLifecycleEnabled = assetLifecycleEnabled;
    }

    /**
     * Returns readable asset list response with pageInfo envelope.
     */
    @GetMapping
    public Map<String, Object> list(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestParam(value = "page", defaultValue = "1") int page,
            @RequestParam(value = "pageSize", defaultValue = "20") int pageSize
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        List<Asset> items = assetService.list(context, page, pageSize);
        long total = assetService.count(context);
        return Map.of(
                "items", items,
                "pageInfo", Map.of(
                        "page", Math.max(page, 1),
                        "pageSize", Math.min(Math.max(pageSize, 1), 200),
                        "total", total
                )
        );
    }

    /**
     * Creates one asset from multipart upload and returns write response envelope.
     */
    @PostMapping(consumes = "multipart/form-data")
    public ResponseEntity<WriteResponse<Asset>> create(
            Authentication authentication,
            HttpServletRequest servletRequest,
            @RequestPart("file") MultipartFile file,
            @RequestParam(value = "name", required = false) String name,
            @RequestParam(value = "type", required = false) String type,
            @RequestParam(value = "mime", required = false) String mime,
            @RequestParam(value = "visibility", required = false) String visibility
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);

        byte[] content;
        try {
            content = file.getBytes();
        } catch (IOException ex) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "file")
            );
        }

        AssetCreateRequest request = new AssetCreateRequest(
                defaultIfBlank(name, file.getOriginalFilename()),
                defaultIfBlank(type, "file"),
                defaultIfBlank(mime, file.getContentType()),
                parseVisibility(visibility)
        );
        return ResponseEntity.accepted().body(assetService.create(request, content, context));
    }

    /**
     * Returns one readable asset or not-found contract envelope.
     */
    @GetMapping("/{assetId}")
    public ResponseEntity<?> get(
            @PathVariable String assetId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        Asset asset = assetService.get(assetId, context);
        if (asset == null) {
            return ResponseEntity.status(404).body(ErrorEnvelope.of("ASSET_NOT_FOUND", "error.asset.not_found"));
        }
        return ResponseEntity.ok(asset);
    }

    /**
     * Updates one asset through command-first domain sugar endpoint.
     */
    @PatchMapping("/{assetId}")
    public ResponseEntity<WriteResponse<Asset>> update(
            @PathVariable String assetId,
            @RequestBody AssetUpdateRequest request,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureLifecycleEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(assetService.update(assetId, request, context));
    }

    /**
     * Deletes one asset through command-first domain sugar endpoint.
     */
    @DeleteMapping("/{assetId}")
    public ResponseEntity<WriteResponse<Asset>> delete(
            @PathVariable String assetId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureLifecycleEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        return ResponseEntity.accepted().body(assetService.delete(assetId, context));
    }

    /**
     * Returns lineage graph for one readable asset.
     */
    @GetMapping("/{assetId}/lineage")
    public ResponseEntity<AssetLineageResponse> lineage(
            @PathVariable String assetId,
            Authentication authentication,
            HttpServletRequest servletRequest
    ) {
        ensureLifecycleEnabled();
        ExecutionContext context = executionContextFactory.resolve(authentication, servletRequest);
        Asset asset = assetService.get(assetId, context);
        if (asset == null) {
            throw ContractException.of(404, "ASSET_NOT_FOUND", "error.asset.not_found");
        }
        return ResponseEntity.ok(assetService.lineage(assetId, context));
    }

    private void ensureLifecycleEnabled() {
        if (!assetLifecycleEnabled) {
            throw ContractException.of(501, "NOT_IMPLEMENTED", "error.asset.not_implemented");
        }
    }

    private String defaultIfBlank(String value, String fallback) {
        if (value == null || value.isBlank()) {
            return fallback == null ? "" : fallback;
        }
        return value.trim();
    }

    private Visibility parseVisibility(String value) {
        if (value == null || value.isBlank()) {
            return Visibility.PRIVATE;
        }
        try {
            return Visibility.valueOf(value.trim().toUpperCase(Locale.ROOT));
        } catch (IllegalArgumentException ex) {
            throw ContractException.of(
                    400,
                    "INVALID_ASSET_REQUEST",
                    "error.asset.invalid_request",
                    Map.of("field", "visibility")
            );
        }
    }
}
