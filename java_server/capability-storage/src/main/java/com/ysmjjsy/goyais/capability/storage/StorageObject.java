/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Storage object descriptor for local/minio/s3 abstraction.
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
