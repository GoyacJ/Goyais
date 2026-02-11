/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Unified object storage SPI for local/minio/s3 providers.
 */
package com.ysmjjsy.goyais.capability.storage;

import java.io.InputStream;

/**
 * Defines object storage SPI that can map to local/minio/s3 providers.
 */
public interface ObjectStorage {

    /**
     * Stores one object and returns persisted metadata.
     */
    StorageObject put(String bucket, String key, InputStream data, String contentType);

    /**
     * Opens one object content stream.
     */
    InputStream get(String bucket, String key);

    /**
     * Deletes one object by bucket and key.
     */
    void delete(String bucket, String key);
}
