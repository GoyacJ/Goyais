/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Repository SPI for asset read/write and permission checks.
 */
package com.ysmjjsy.goyais.application.asset;

import com.ysmjjsy.goyais.contract.api.common.Asset;
import com.ysmjjsy.goyais.contract.api.common.AssetLineageEdge;
import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Visibility;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;
import java.util.Map;

/**
 * Persists and queries assets with SQL-layer permission semantics.
 */
public interface AssetRepository {

    /**
     * Creates one asset row and returns the persisted resource.
     */
    Asset create(
            ExecutionContext context,
            String name,
            String type,
            String mime,
            long size,
            String hash,
            String uri,
            Visibility visibility,
            Map<String, Object> metadata,
            Instant now
    );

    /**
     * Returns one in-scope asset regardless of ACL read filtering.
     */
    Asset findByIdInScope(String assetId, ExecutionContext context);

    /**
     * Returns one readable asset or null when inaccessible.
     */
    Asset findReadableById(String assetId, ExecutionContext context);

    /**
     * Returns readable assets ordered by created_at DESC and id DESC.
     */
    List<Asset> listReadable(ExecutionContext context, int page, int pageSize);

    /**
     * Returns count of readable assets for current context.
     */
    long countReadable(ExecutionContext context);

    /**
     * Updates mutable fields and returns latest persisted asset.
     */
    Asset update(
            String assetId,
            ExecutionContext context,
            String name,
            Visibility visibility,
            Map<String, Object> metadata,
            boolean metadataProvided,
            Instant now
    );

    /**
     * Marks one asset as deleted and returns latest persisted resource.
     */
    Asset markDeleted(String assetId, ExecutionContext context, Instant now);

    /**
     * Checks one permission against user/role ACL grants.
     */
    boolean hasPermission(String assetId, ExecutionContext context, Permission permission, Instant now);

    /**
     * Returns lineage edges for one asset in current scope.
     */
    List<AssetLineageEdge> listLineage(String assetId, ExecutionContext context);
}
