/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Storage object descriptor for local/minio/s3 abstraction.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.storage;

/**
 * Represents normalized object metadata returned by storage providers.
 */
public record StorageObject(
        String bucket,
        String key,
        String contentType,
        long size
) {
}
