/**
 * SPDX-License-Identifier: Apache-2.0
 * Copyright (c) 2026 Goya
 * Author: Goya
 * Created: 2026-02-11
 * Version: v1.0.0
 * Description: Goyais source file.
 */

import type { Visibility } from '@/design-system/types'

export interface MockAsset {
  assetId: string
  name: string
  type: string
  size: string
  visibility: Visibility
  createdAt: string
  uri: string
  hash: string
  owner: string
}

export const mockAssets: MockAsset[] = [
  {
    assetId: 'ast_01hz9p0a2rfk9dnf0xj8s3q9y1',
    name: 'warehouse-cam-01.mp4',
    type: 'video/mp4',
    size: '82.4 MB',
    visibility: 'WORKSPACE',
    createdAt: '2026-02-10T09:02:11Z',
    uri: 's3://workspace-alpha/warehouse-cam-01.mp4',
    hash: 'sha256:8bd8397ce91af0fe71f4bb6a7de11f3a',
    owner: 'u_alice',
  },
  {
    assetId: 'ast_01hz9p0b8k2x3j2v1wvj9mv7r4',
    name: 'dashboard-snapshot.png',
    type: 'image/png',
    size: '3.1 MB',
    visibility: 'PRIVATE',
    createdAt: '2026-02-10T08:41:06Z',
    uri: 's3://workspace-alpha/dashboard-snapshot.png',
    hash: 'sha256:ab924f78f53d6a6e29529bf00c9b5f23',
    owner: 'u_bob',
  },
  {
    assetId: 'ast_01hz9p0cw81an3mk4x2bqxg5e6',
    name: 'daily-report.json',
    type: 'application/json',
    size: '468 KB',
    visibility: 'TENANT',
    createdAt: '2026-02-09T23:13:52Z',
    uri: 's3://tenant-reports/daily-report.json',
    hash: 'sha256:1ff8cfa2828f0fdb2e3f6fe23aa3f3d1',
    owner: 'u_ops',
  },
]
