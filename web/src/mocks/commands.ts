import type { CommandStatus } from '@/design-system/types'

export interface MockCommand {
  commandId: string
  commandType: string
  status: CommandStatus
  acceptedAt: string
  startedAt: string
  finishedAt?: string
  resultSummary: string
  logs: string[]
}

export const mockCommands: MockCommand[] = [
  {
    commandId: 'cmd_01hzy8w6hqvhm2k6d9w6c2c4x1',
    commandType: 'workflow.run',
    status: 'running',
    acceptedAt: '2026-02-10T10:02:13Z',
    startedAt: '2026-02-10T10:02:14Z',
    resultSummary: 'Dispatching 3 steps...',
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
    logs: [
      '[09:12:02.120] validating source path',
      '[09:12:08.983] error.command.invalidPayload',
      '[09:12:10.044] command marked failed',
    ],
  },
]
