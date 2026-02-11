import type { CommandStatus } from '@/design-system/types'

export interface MockCommand {
  commandId: string
  commandType: string
  status: CommandStatus
  acceptedAt: string
  startedAt: string
  finishedAt?: string
  resultSummary: string
  owner: string
  traceId: string
  logs: string[]
}

export const mockCommands: MockCommand[] = [
  {
    commandId: 'cmd_01hzy8w4h9f8n3rf7c8m8w0v21',
    commandType: 'workflow.run',
    status: 'accepted',
    acceptedAt: '2026-02-10T10:01:13Z',
    startedAt: '2026-02-10T10:01:14Z',
    resultSummary: 'Queued at command gate.',
    owner: 'u_alice',
    traceId: 'tr_8n2f3a01',
    logs: [
      '[10:01:13.012] accepted by command gate',
      '[10:01:13.025] waiting for worker slot',
    ],
  },
  {
    commandId: 'cmd_01hzy8w6hqvhm2k6d9w6c2c4x1',
    commandType: 'workflow.run',
    status: 'running',
    acceptedAt: '2026-02-10T10:02:13Z',
    startedAt: '2026-02-10T10:02:14Z',
    resultSummary: 'Dispatching 3 steps.',
    owner: 'u_alice',
    traceId: 'tr_8n2f3a02',
    logs: [
      '[10:02:14.012] accepted by command gate',
      '[10:02:14.517] step normalize.input started',
      '[10:02:15.381] step normalize.input completed',
      '[10:02:15.492] step render.preview started',
    ],
  },
  {
    commandId: 'cmd_01hzy8w8p2tvxryvzz6p6yxq7j',
    commandType: 'plugin.install',
    status: 'succeeded',
    acceptedAt: '2026-02-10T09:44:20Z',
    startedAt: '2026-02-10T09:44:21Z',
    finishedAt: '2026-02-10T09:44:49Z',
    resultSummary: 'Plugin package verified and enabled.',
    owner: 'u_bob',
    traceId: 'tr_8n2f3a03',
    logs: [
      '[09:44:21.031] package checksum verified',
      '[09:44:35.907] dependency graph resolved',
      '[09:44:49.145] install completed',
    ],
  },
  {
    commandId: 'cmd_01hzy8wa8gmvxa6r4f3r7k9m16',
    commandType: 'stream.record.start',
    status: 'failed',
    acceptedAt: '2026-02-10T09:12:01Z',
    startedAt: '2026-02-10T09:12:02Z',
    finishedAt: '2026-02-10T09:12:10Z',
    resultSummary: 'Missing source endpoint in payload.',
    owner: 'u_carol',
    traceId: 'tr_8n2f3a04',
    logs: [
      '[09:12:02.120] validating source path',
      '[09:12:08.983] error.command.invalidPayload',
      '[09:12:10.044] command marked failed',
    ],
  },
  {
    commandId: 'cmd_01hzy8wbne2fnm9m6p2t9rgc2a',
    commandType: 'workflow.cancel',
    status: 'canceled',
    acceptedAt: '2026-02-10T08:20:17Z',
    startedAt: '2026-02-10T08:20:18Z',
    finishedAt: '2026-02-10T08:20:19Z',
    resultSummary: 'Canceled by owner request.',
    owner: 'u_alice',
    traceId: 'tr_8n2f3a05',
    logs: [
      '[08:20:18.001] cancel requested by user',
      '[08:20:18.415] stop signal propagated',
      '[08:20:19.002] run marked canceled',
    ],
  },
]
