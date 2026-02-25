import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, writeFileSync } from 'node:fs'
import test from 'node:test'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)
const repoRoot = resolve(__dirname, '..', '..')
const scriptPath = join(repoRoot, 'scripts', 'refactor', 't001-capture-kode-cli-baseline.mjs')
const referenceRoot = join(repoRoot, 'Kode-cli')

function runScript(args) {
  return spawnSync(process.execPath, [scriptPath, ...args], {
    cwd: repoRoot,
    encoding: 'utf8',
  })
}

test('captures and validates a temporary baseline', () => {
  const tmpDir = mkdtempSync(join(tmpdir(), 'goyais-t001-test-'))
  const outPath = join(tmpDir, 'baseline.json')

  const capture = runScript(['--reference', referenceRoot, '--out', outPath])
  assert.equal(capture.status, 0, capture.stderr)

  const baseline = JSON.parse(readFileSync(outPath, 'utf8'))
  assert.equal(baseline.task_id, 'T-001')
  assert.equal(Array.isArray(baseline.cases), true)
  assert.equal(baseline.cases.length >= 4, true)

  const check = runScript(['--check', '--reference', referenceRoot, '--out', outPath])
  assert.equal(check.status, 0, check.stderr)
})

test('fails check when baseline artifact is stale', () => {
  const tmpDir = mkdtempSync(join(tmpdir(), 'goyais-t001-test-stale-'))
  const outPath = join(tmpDir, 'baseline.json')

  const capture = runScript(['--reference', referenceRoot, '--out', outPath])
  assert.equal(capture.status, 0, capture.stderr)

  const baseline = JSON.parse(readFileSync(outPath, 'utf8'))
  baseline.cases[0].exit_code = baseline.cases[0].exit_code === 0 ? 9 : 0
  writeFileSync(outPath, `${JSON.stringify(baseline, null, 2)}\n`)

  const check = runScript(['--check', '--reference', referenceRoot, '--out', outPath])
  assert.equal(check.status, 1)
})
