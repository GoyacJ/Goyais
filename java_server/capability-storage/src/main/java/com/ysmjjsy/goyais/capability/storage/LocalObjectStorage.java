/**
 * SPDX-License-Identifier: Apache-2.0
 * <p>Local filesystem object storage provider for minimal profile.</p>
 * @author Goya
 * @since 2026-02-12 01:20:09
 */

package com.ysmjjsy.goyais.capability.storage;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;

public final class LocalObjectStorage implements ObjectStorage {
    private final Path root;

    /**
     * Creates local storage provider rooted at the supplied filesystem path.
     * @param root TODO
     */
    public LocalObjectStorage(Path root) {
        this.root = root;
    }

    /**
     * Writes uploaded data to local filesystem and returns descriptor metadata.
     * @param bucket TODO
     * @param key TODO
     * @param data TODO
     * @param contentType TODO
     * @return TODO
     */
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

    /**
     * Reads one object stream from local filesystem.
     * @param bucket TODO
     * @param key TODO
     * @return TODO
     */
    @Override
    public InputStream get(String bucket, String key) {
        try {
            return Files.newInputStream(root.resolve(bucket).resolve(key));
        } catch (IOException ex) {
            throw new IllegalStateException("failed to read local object", ex);
        }
    }

    /**
     * Deletes one object from local filesystem if it exists.
     * @param bucket TODO
     * @param key TODO
     */
    @Override
    public void delete(String bucket, String key) {
        try {
            Files.deleteIfExists(root.resolve(bucket).resolve(key));
        } catch (IOException ex) {
            throw new IllegalStateException("failed to delete local object", ex);
        }
    }
}
