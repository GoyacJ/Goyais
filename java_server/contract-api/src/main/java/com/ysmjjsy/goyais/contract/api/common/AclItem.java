/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>ACL item contract model.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.contract.api.common;

import java.time.Instant;
import java.util.Set;

/**
 * Describes one ACL grant entry in API resource representations.
 */
public record AclItem(
        String subjectType,
        String subjectId,
        Set<Permission> permissions,
        Instant expiresAt
) {
}
