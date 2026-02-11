/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Repository SPI for share grant persistence and permission checks.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.application.share;

import com.ysmjjsy.goyais.contract.api.common.Permission;
import com.ysmjjsy.goyais.contract.api.common.Share;
import com.ysmjjsy.goyais.contract.api.common.ShareCreateRequest;
import com.ysmjjsy.goyais.kernel.core.ExecutionContext;
import java.time.Instant;
import java.util.List;

/**
 * Persists ACL share grants and exposes share authorization lookups.
 */
public interface ShareRepository {

    /**
     * Creates one share grant row and returns the persisted share resource.
     */
    Share create(ShareCreateRequest request, ExecutionContext context, Instant now);

    /**
     * Returns shares in current tenant/workspace sorted by newest first.
     */
    List<Share> list(ExecutionContext context, int page, int pageSize);

    /**
     * Returns share count in current tenant/workspace.
     */
    long count(ExecutionContext context);

    /**
     * Deletes one share created by current user and returns true when deleted.
     */
    boolean deleteByIdAndCreator(String shareId, ExecutionContext context);

    /**
     * Returns resource scope metadata used by share.create authorization.
     */
    ShareResourceScope findResourceScope(String resourceType, String resourceId, ExecutionContext context);

    /**
     * Checks whether current subject has the requested permission on resource.
     */
    boolean hasResourcePermission(
            String resourceType,
            String resourceId,
            ExecutionContext context,
            Permission permission,
            Instant now
    );
}
