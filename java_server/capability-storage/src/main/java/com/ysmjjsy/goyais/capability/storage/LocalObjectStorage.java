/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Local filesystem object storage provider for minimal profile.
 */
package com.ysmjjsy.goyais.capability.storage;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;

public final class LocalObjectStorage implements ObjectStorage {
    private final Path root;

    public LocalObjectStorage(Path root) {
        this.root = root;
    }

    @Override
    public StorageObject put(String bucket, String key, InputStream data, String contentType) {
        try {
            Path target = root.resolve(bucket).resolve(key);
            Files.createDirectories(target.getParent());
            long size = Files.copy(data, target, StandardCopyOption.REPLACE_EXISTING);
            return new StorageObject(bucket, key, contentType, size);
        } catch (IOException ex) {
            throw new IllegalStateException("failed to write local object", ex);
        }
    }

    @Override
    public InputStream get(String bucket, String key) {
        try {
            return Files.newInputStream(root.resolve(bucket).resolve(key));
        } catch (IOException ex) {
            throw new IllegalStateException("failed to read local object", ex);
        }
    }

    @Override
    public void delete(String bucket, String key) {
        try {
            Files.deleteIfExists(root.resolve(bucket).resolve(key));
        } catch (IOException ex) {
            throw new IllegalStateException("failed to delete local object", ex);
        }
    }
}
