#!/usr/bin/env node
// Update or verify every place that carries the application version.
//
// Usage:
//   node scripts/update-version.mjs 1.4.0        # write the new version
//   node scripts/update-version.mjs --check      # verify VERSION everywhere
//   node scripts/update-version.mjs --check 1.4.0
//
// VERSION is the single source of truth. The same value is propagated to the
// Go fallback, frontend package metadata, Wails product metadata and the README
// version block. Release builds additionally inject it into the binary with
// -ldflags "-X main.appVersion=<version>".

import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const SEMVER = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/
const args = process.argv.slice(2)
const check = args.includes("--check")
const explicit = args.find((arg) => !arg.startsWith("--"))

const versionFile = path.join(root, "VERSION")
const current = fs.existsSync(versionFile) ? fs.readFileSync(versionFile, "utf8").trim() : ""
const target = explicit || current

const problems = []
const changed = []

function fail(message) {
  console.error(`error: ${message}`)
  process.exit(1)
}

function read(rel) {
  return fs.readFileSync(path.join(root, rel), "utf8")
}

function write(rel, content) {
  fs.writeFileSync(path.join(root, rel), content)
  changed.push(rel)
}

function readJSON(rel) {
  try {
    return JSON.parse(read(rel))
  } catch (error) {
    fail(`${rel} is not valid JSON: ${error.message}`)
  }
}

function writeJSON(rel, value) {
  write(rel, `${JSON.stringify(value, null, 2)}\n`)
}

function expect(rel, actual, wanted, apply) {
  if (actual === wanted) return
  problems.push(`${rel}: ${JSON.stringify(actual)} (expected ${JSON.stringify(wanted)})`)
  if (!check) apply()
}

if (!SEMVER.test(target)) {
  fail(`invalid version ${JSON.stringify(target)}; expected MAJOR.MINOR.PATCH (for example 1.4.0)`)
}

// 1. VERSION file.
expect("VERSION", current, target, () => write("VERSION", `${target}\n`))

// 2. Go fallback used by local `go run`/`go build` without release ldflags.
const appRel = "app.go"
const appSource = read(appRel)
const appMatch = appSource.match(/(?:const|var)\s+appVersion\s*=\s*"([^"]*)"/)
if (!appMatch) fail("app.go does not declare appVersion")
expect("app.go appVersion", appMatch[1], target, () =>
  write(appRel, appSource.replace(appMatch[0], `var appVersion = "${target}"`)),
)

// 3. Frontend package metadata.
const pkgRel = "frontend/package.json"
const pkg = readJSON(pkgRel)
expect(pkgRel, pkg.version, target, () => {
  pkg.version = target
  writeJSON(pkgRel, pkg)
})

const lockRel = "frontend/package-lock.json"
const lock = readJSON(lockRel)
expect(`${lockRel} root version`, lock.version, target, () => {
  lock.version = target
  if (lock.packages && lock.packages[""]) lock.packages[""].version = target
  writeJSON(lockRel, lock)
})
if (!lock.packages || !lock.packages[""]) {
  fail("frontend/package-lock.json is missing packages[\"\"] metadata")
}
expect(`${lockRel} packages[\"\"]`, lock.packages[""].version, target, () => {
  lock.packages[""].version = target
  writeJSON(lockRel, lock)
})

// 4. Wails product metadata (drives the Windows executable and NSIS installer).
const wailsRel = "wails.json"
const wails = readJSON(wailsRel)
const wailsVersion = wails.info && wails.info.productVersion
expect(`${wailsRel} info.productVersion`, wailsVersion, target, () => {
  wails.info = wails.info || {}
  wails.info.productVersion = target
  writeJSON(wailsRel, wails)
})

// 5. Windows resources keep templates so Wails resolves the version at build.
for (const rel of ["build/windows/info.json", "build/windows/wails.exe.manifest"]) {
  if (!read(rel).includes("{{.Info.ProductVersion}}")) {
    problems.push(`${rel}: missing {{.Info.ProductVersion}} template`)
  }
}

// 6. README version block.
const readmeRel = "README.md"
const readme = read(readmeRel)
const marker = /<!-- version:start -->[\s\S]*?<!-- version:end -->/
const markerMatch = readme.match(marker)
if (!markerMatch) {
  problems.push(`${readmeRel}: missing <!-- version:start --> ... <!-- version:end --> block`)
} else {
  const shown = markerMatch[0].match(/`([^`]+)`/)
  expect(`${readmeRel} version block`, shown ? shown[1] : "", target, () =>
    write(
      readmeRel,
      readme.replace(
        marker,
        `<!-- version:start -->\n**Current version:** \`${target}\`\n<!-- version:end -->`,
      ),
    ),
  )
}

if (check) {
  if (problems.length > 0) {
    console.error("Version metadata is inconsistent:\n- " + problems.join("\n- "))
    process.exit(1)
  }
  console.log(`Version ${target} is consistent across the repository.`)
  process.exit(0)
}

if (problems.length === 0) {
  console.log(`Version ${target} is already applied everywhere.`)
} else {
  console.log(`Updated version to ${target}:`)
  for (const rel of [...new Set(changed)]) console.log(`  - ${rel}`)
}