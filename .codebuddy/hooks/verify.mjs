#!/usr/bin/env node
/**
 * CodeBuddy PostToolUse 钩子：改动源码后自动运行对应验证命令。
 *
 * - 改到 *.go：go vet ./...（编译 + 静态检查）
 * - 改到 ui/ 下的 vue / ts / css：npm run type-check（vue-tsc）
 *
 * 校验失败时退出码 2，把错误摘要通过 stderr 回传给 Agent，让它接着修；
 * 通过时退出码 0，输出一行提示。同一时间只跑一次（用锁文件防重复），
 * 避免一次响应里改好几个文件时并发跑多次类型检查。
 */
import { execSync } from 'node:child_process'
import { readFileSync, statSync, unlinkSync, writeFileSync } from 'node:fs'
import { dirname, join, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const hookDir = dirname(fileURLToPath(import.meta.url))
const projectDir =
  process.env.CODEBUDDY_PROJECT_DIR || process.env.CLAUDE_PROJECT_DIR || resolve(hookDir, '..', '..')
const uiDir = join(projectDir, 'ui')

/** 锁文件：有校验正在跑（或刚跑完）时跳过，防止并行重复执行 */
const LOCK_FILE = join(hookDir, '.verify.lock')
const LOCK_TTL = 120 * 1000
/** 回传给 Agent 的错误输出上限，避免刷屏 */
const MAX_OUTPUT = 3000

const readStdin = () => {
  try {
    // 去掉 BOM 与首尾空白（部分终端/管道会给 stdin 带上 BOM，JSON.parse 会因此失败）
    return readFileSync(0, 'utf8').replace(/^\uFEFF+/, '').trim()
  } catch {
    return ''
  }
}

/** 正在校验 / 刚校验完（锁没过期）时返回 true */
const isLocked = () => {
  try {
    return Date.now() - statSync(LOCK_FILE).mtimeMs < LOCK_TTL
  } catch {
    return false
  }
}

/** 运行一条验证命令；通过返回 null，失败返回错误摘要 */
const run = (command, cwd, label) => {
  try {
    execSync(command, {
      cwd,
      stdio: ['ignore', 'pipe', 'pipe'],
      windowsHide: true,
      env: process.env,
    })
    return null
  } catch (err) {
    const output = `${err.stdout || ''}${err.stderr || ''}`.toString().trim()
    return { label, output: output.slice(0, MAX_OUTPUT) }
  }
}

const main = () => {
  const payload = (() => {
    try {
      return JSON.parse(readStdin() || '{}')
    } catch {
      return {}
    }
  })()

  const input = payload.tool_input || {}
  const filePath = String(input.file_path || input.filePath || input.path || '')
  if (!filePath) {
    return 0
  }

  const rel = relative(projectDir, resolve(projectDir, filePath)).replace(/\\/g, '/')
  // 构建产物、依赖、运行时数据不参与校验
  if (
    !rel ||
    rel.startsWith('..') ||
    rel.startsWith('public/') ||
    rel.startsWith('data/') ||
    rel.includes('node_modules/') ||
    rel.endsWith('package-lock.json')
  ) {
    return 0
  }

  const needGo = rel.endsWith('.go')
  const needUi = rel.startsWith('ui/') && /\.(vue|ts|tsx|js|jsx|css|scss)$/.test(rel)
  if (!needGo && !needUi) {
    return 0
  }

  if (isLocked()) {
    return 0
  }

  writeFileSync(LOCK_FILE, String(Date.now()))
  const failures = []
  try {
    if (needGo) {
      const failure = run('go vet ./...', projectDir, 'go vet ./...')
      if (failure) failures.push(failure)
    }
    if (needUi) {
      const failure = run('npm run type-check', uiDir, 'npm run type-check')
      if (failure) failures.push(failure)
    }
  } finally {
    try {
      unlinkSync(LOCK_FILE)
    } catch {
      /* 锁文件已被清理：忽略 */
    }
  }

  if (!failures.length) {
    console.log(`自动验证通过：${[needGo && 'go vet', needUi && 'type-check'].filter(Boolean).join(' / ')}`)
    return 0
  }

  const message = failures
    .map((failure) => `❌ ${failure.label}\n${failure.output || '（无输出）'}`)
    .join('\n\n')
  console.error(`自动验证未通过，请修正后重试：\n\n${message}`)
  return 2
}

try {
  process.exitCode = main()
} catch (err) {
  console.error(`自动验证脚本异常：${err && err.message ? err.message : err}`)
  process.exitCode = 2
}
