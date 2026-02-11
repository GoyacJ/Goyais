/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Unified object storage SPI for local/minio/s3 providers.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
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
