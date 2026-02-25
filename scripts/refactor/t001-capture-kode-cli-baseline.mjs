#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import {
  copyFileSync,
  mkdtempSync,
  mkdirSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const SCHEMA_VERSION = 1
const TASK_ID = 'T-001'
const BASELINE_KIND = 'kode-cli-wrapper-bootstrap'

const CASES = [
  {
    id: 'cli_help_lite',
    entrypoint: 'cli.js',
    args: ['--help-lite'],
    description: 'Wrapper help-lite usage contract',
  },
  {
    id: 'cli_version',
    entrypoint: 'cli.js',
    args: ['--version'],
    description: 'Wrapper version output contract',
  },
  {
    id: 'cli_help_without_dist',
    entrypoint: 'cli.js',
    args: ['--help'],
    description: 'Bootstrap fallback when dist/native binary is unavailable',
  },
  {
    id: 'acp_without_dist',
    entrypoint: 'cli-acp.js',
    args: [],
    description: 'ACP bootstrap fallback when dist/native binary is unavailable',
  },
]

function usage() {
  console.log(
    [
      'Usage:',
      '  node scripts/refactor/t001-capture-kode-cli-baseline.mjs [--check] [--reference <path>] [--out <file>]',
      '',
      'Options:',
      '  --check             Validate current outputs against the golden baseline',
      '  --reference <path>  Kode-cli reference repo root (default: <repo>/Kode-cli)',
      '  --out <file>        Golden JSON output path (default: docs/refactor/parity/kode-cli-wrapper-golden.json)',
    ].join('\n'),
  )
}

function normalizeNewlines(text) {
  return String(text ?? '').replace(/\r\n/g, '\n')
}

function parseArgs(argv) {
  const parsed = {
    check: false,
    reference: null,
    out: null,
  }

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    if (arg === '--check') {
      parsed.check = true
      continue
    }
    if (arg === '--reference') {
      const value = argv[i + 1]
      if (!value) {
        throw new Error('--reference requires a path value')
      }
      parsed.reference = value
      i++
      continue
    }
    if (arg === '--out') {
      const value = argv[i + 1]
      if (!value) {
        throw new Error('--out requires a file path value')
      }
      parsed.out = value
      i++
      continue
    }
    if (arg === '--help' || arg === '-h') {
      usage()
      process.exit(0)
    }

    throw new Error(`Unknown argument: ${arg}`)
  }

  return parsed
}

function resolveGitCommit(referenceRoot) {
  const result = spawnSync('git', ['-C', referenceRoot, 'rev-parse', 'HEAD'], {
    encoding: 'utf8',
  })
  if (result.status !== 0) {
    return 'unknown'
  }
  return String(result.stdout ?? '').trim() || 'unknown'
}

function readPackageVersion(referenceRoot) {
  const pkgPath = join(referenceRoot, 'package.json')
  const pkg = JSON.parse(readFileSync(pkgPath, 'utf8'))
  return String(pkg.version ?? 'unknown')
}

function runCase(referenceRoot, projectCwd, env, testCase) {
  const cmdArgs = [join(referenceRoot, testCase.entrypoint), ...testCase.args]
  const result = spawnSync(process.execPath, cmdArgs, {
    cwd: projectCwd,
    env,
    encoding: 'utf8',
    timeout: 30_000,
    maxBuffer: 8 * 1024 * 1024,
  })

  return {
    id: testCase.id,
    description: testCase.description,
    argv: [testCase.entrypoint, ...testCase.args],
    exit_code: typeof result.status === 'number' ? result.status : 1,
    stdout: normalizeNewlines(result.stdout ?? ''),
    stderr: normalizeNewlines(result.stderr ?? ''),
  }
}

function createMinimalReferenceCopy(referenceRoot, scratchRoot) {
  const copiedRoot = join(scratchRoot, 'reference')
  mkdirSync(join(copiedRoot, 'scripts'), { recursive: true })
  copyFileSync(join(referenceRoot, 'cli.js'), join(copiedRoot, 'cli.js'))
  copyFileSync(join(referenceRoot, 'cli-acp.js'), join(copiedRoot, 'cli-acp.js'))
  copyFileSync(join(referenceRoot, 'package.json'), join(copiedRoot, 'package.json'))
  copyFileSync(
    join(referenceRoot, 'scripts', 'binary-utils.cjs'),
    join(copiedRoot, 'scripts', 'binary-utils.cjs'),
  )
  return copiedRoot
}

function capture(referenceRoot, referenceLabel) {
  const tmpRoot = mkdtempSync(join(tmpdir(), 'goyais-t001-parity-'))
  const configDir = join(tmpRoot, 'config')
  const binDir = join(tmpRoot, 'bin')
  const projectCwd = join(tmpRoot, 'project')
  const copiedReferenceRoot = createMinimalReferenceCopy(referenceRoot, tmpRoot)

  mkdirSync(configDir, { recursive: true })
  mkdirSync(binDir, { recursive: true })
  mkdirSync(projectCwd, { recursive: true })

  writeFileSync(
    join(configDir, 'config.json'),
    JSON.stringify(
      {
        theme: 'dark',
        hasCompletedOnboarding: true,
        preferredNotifChannel: 'iterm2',
        verbose: false,
        numStartups: 0,
      },
      null,
      2,
    ),
  )

  const env = {
    ...process.env,
    NODE_ENV: 'test',
    KODE_CONFIG_DIR: configDir,
    CLAUDE_CONFIG_DIR: configDir,
    ANYKODE_CONFIG_DIR: configDir,
    KODE_BIN_DIR: binDir,
    ANYKODE_BIN_DIR: binDir,
  }

  const cases = CASES.map(testCase =>
    runCase(copiedReferenceRoot, projectCwd, env, testCase),
  )

  rmSync(tmpRoot, { recursive: true, force: true })

  return {
    schema_version: SCHEMA_VERSION,
    task_id: TASK_ID,
    baseline_kind: BASELINE_KIND,
    captured_at_utc: new Date().toISOString(),
    reference: {
      path: referenceLabel,
      git_commit: resolveGitCommit(referenceRoot),
      package_version: readPackageVersion(referenceRoot),
    },
    cases,
  }
}

function comparableBaseline(baseline) {
  return {
    schema_version: baseline.schema_version,
    task_id: baseline.task_id,
    baseline_kind: baseline.baseline_kind,
    reference: baseline.reference,
    cases: baseline.cases,
  }
}

function writeBaseline(outPath, baseline) {
  mkdirSync(dirname(outPath), { recursive: true })
  writeFileSync(outPath, `${JSON.stringify(baseline, null, 2)}\n`)
}

function readBaseline(outPath) {
  return JSON.parse(readFileSync(outPath, 'utf8'))
}

function findCaseByID(cases, id) {
  return cases.find(item => item.id === id) || null
}

function printCaseMismatch(expectedCase, actualCase) {
  console.error(`- case: ${expectedCase.id}`)
  if (expectedCase.exit_code !== actualCase.exit_code) {
    console.error(
      `  exit_code expected=${expectedCase.exit_code} actual=${actualCase.exit_code}`,
    )
  }
  if (expectedCase.stdout !== actualCase.stdout) {
    console.error('  stdout mismatch')
  }
  if (expectedCase.stderr !== actualCase.stderr) {
    console.error('  stderr mismatch')
  }
}

function checkBaseline(expected, actual) {
  const expectedCore = comparableBaseline(expected)
  const actualCore = comparableBaseline(actual)

  const expectedCases = expectedCore.cases
  const actualCases = actualCore.cases

  const metadataMatches =
    expectedCore.schema_version === actualCore.schema_version &&
    expectedCore.task_id === actualCore.task_id &&
    expectedCore.baseline_kind === actualCore.baseline_kind &&
    JSON.stringify(expectedCore.reference) === JSON.stringify(actualCore.reference)

  if (!metadataMatches) {
    console.error('Baseline metadata mismatch.')
    console.error(`expected reference: ${JSON.stringify(expectedCore.reference)}`)
    console.error(`actual reference:   ${JSON.stringify(actualCore.reference)}`)
    return false
  }

  if (expectedCases.length !== actualCases.length) {
    console.error(
      `Case count mismatch: expected ${expectedCases.length}, actual ${actualCases.length}`,
    )
    return false
  }

  let ok = true
  for (const expectedCase of expectedCases) {
    const actualCase = findCaseByID(actualCases, expectedCase.id)
    if (!actualCase) {
      console.error(`Missing case in actual output: ${expectedCase.id}`)
      ok = false
      continue
    }
    if (
      expectedCase.exit_code !== actualCase.exit_code ||
      expectedCase.stdout !== actualCase.stdout ||
      expectedCase.stderr !== actualCase.stderr
    ) {
      printCaseMismatch(expectedCase, actualCase)
      ok = false
    }
  }

  return ok
}

function main() {
  const __filename = fileURLToPath(import.meta.url)
  const __dirname = dirname(__filename)
  const repoRoot = resolve(__dirname, '..', '..')

  let args
  try {
    args = parseArgs(process.argv.slice(2))
  } catch (error) {
    console.error(String(error))
    usage()
    process.exit(1)
  }
  const referenceRoot = resolve(args.reference ?? join(repoRoot, 'Kode-cli'))
  const referenceLabelRaw = relative(repoRoot, referenceRoot).replaceAll('\\', '/')
  const referenceLabel =
    referenceLabelRaw && !referenceLabelRaw.startsWith('..')
      ? referenceLabelRaw
      : referenceRoot
  const outPath = resolve(
    args.out ?? join(repoRoot, 'docs', 'refactor', 'parity', 'kode-cli-wrapper-golden.json'),
  )

  const actual = capture(referenceRoot, referenceLabel)

  if (args.check) {
    let expected
    try {
      expected = readBaseline(outPath)
    } catch (error) {
      console.error(`Failed to read golden baseline: ${outPath}`)
      console.error(String(error))
      process.exit(1)
    }

    const ok = checkBaseline(expected, actual)
    if (!ok) {
      console.error('Parity check failed. Re-capture baseline if changes are intentional:')
      console.error(
        '  node scripts/refactor/t001-capture-kode-cli-baseline.mjs --reference <path>',
      )
      process.exit(1)
    }

    console.log(`Parity check passed: ${outPath}`)
    process.exit(0)
  }

  writeBaseline(outPath, actual)
  console.log(`Baseline captured: ${outPath}`)
}

main()
