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

public interface ObjectStorage {
    StorageObject put(String bucket, String key, InputStream data, String contentType);

    InputStream get(String bucket, String key);

    void delete(String bucket, String key);
}
