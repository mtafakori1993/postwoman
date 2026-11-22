#!/usr/bin/env node
// Generate categorized release notes and a changelog entry from git history.
//
// Usage:
//   node scripts/generate-release-notes.mjs 1.4.0
//   node scripts/generate-release-notes.mjs 1.4.0 --from v1.3.3 --out release-notes.md --no-changelog
//   node scripts/generate-release-notes.mjs 1.4.0 --stdout
//
// Only commit subjects are summarised; raw commit dumps are never emitted.
// Conventional commit prefixes drive the sections, and every bullet links to
// its commit. Commits without a recognised prefix fall into Technical Changes.

import { execFileSync } from "node:child_process"
import fs from "node:fs"
import path from "node:path"
import { fileURLToPath } from "node:url"

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..")
const repo = process.env.GITHUB_REPOSITORY || "mtafakori1993/postwoman"
const repoURL = `https://github.com/${repo}`
const SEMVER = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/

const args = process.argv.slice(2)
const options = {
  from: "",
  out: "",
  changelog: "CHANGELOG.md",
  noChangelog: false,
  stdout: false,
}
const positional = []
for (let i = 0; i < args.length; i += 1) {
  const arg = args[i]
  if (arg === "--no-changelog") options.noChangelog = true
  else if (arg === "--stdout") options.stdout = true
  else if (arg === "--from") options.from = args[++i] || ""
  else if (arg.startsWith("--from=")) options.from = arg.slice(7)
  else if (arg === "--out") options.out = args[++i] || ""
  else if (arg.startsWith("--out=")) options.out = arg.slice(6)
  else if (arg === "--changelog") options.changelog = args[++i] || "CHANGELOG.md"
  else if (arg.startsWith("--changelog=")) options.changelog = arg.slice(12)
  else if (!arg.startsWith("--")) positional.push(arg)
}

const version = positional[0]
if (!version) {
  console.error("usage: node scripts/generate-release-notes.mjs <version> [--from <tag>] [--out <file>] [--no-changelog]")
  process.exit(1)
}
if (!SEMVER.test(version)) {
  console.error(`error: invalid version ${JSON.stringify(version)}; expected MAJOR.MINOR.PATCH`)
  process.exit(1)
}

function git(gitArgs) {
  return execFileSync("git", gitArgs, { cwd: root, encoding: "utf8" }).trim()
}

function previousTag() {
  if (options.from) return options.from
  const tags = git(["tag", "--merged", "HEAD", "--list", "v[0-9]*", "--sort=-v:refname"])
    .split("\n")
    .map((tag) => tag.trim())
    .filter((tag) => tag && tag !== `v${version}`)
  return tags[0] || ""
}

function commits(range) {
  const raw = git(["log", "--no-merges", "--format=%H%x1f%s%x1f%b%x1e", range])
  if (!raw) return []
  return raw
    .split("\x1e")
    .map((record) => record.trim())
    .filter(Boolean)
    .map((record) => {
      const [hash, subject, body = ""] = record.split("\x1f")
      return { hash: hash.trim(), subject: subject.trim(), body }
    })
    .filter((commit) => commit.hash && commit.subject)
}

function classify(commit) {
  const { subject, body } = commit
  if (/^chore\(release\)/i.test(subject) || /^release:/i.test(subject)) return null
  if (/^merge\b/i.test(subject)) return null
  const match = subject.match(/^([a-zA-Z]+)(?:\(([^)]+)\))?(!)?:\s*(.+)$/)
  const type = match ? match[1].toLowerCase() : ""
  const scope = match && match[2] ? match[2] : ""
  const bang = !!(match && match[3])
  const message = (match ? match[4] : subject).trim()
  if (!message) return null
  const breaking = bang || /BREAKING[ -]CHANGE:/i.test(body)
  let section = "technical"
  if (type === "feat") section = "features"
  else if (type === "fix") section = "fixes"
  else if (type === "perf") section = "improvements"
  return {
    section,
    breaking,
    text: message.charAt(0).toUpperCase() + message.slice(1).replace(/[.]+$/, ""),
    scope,
    hash: commit.hash.slice(0, 7),
  }
}

function bullet(commit) {
  const scope = commit.scope ? `**${commit.scope}:** ` : ""
  return `${scope}${commit.text} ([${commit.hash}](${repoURL}/commit/${commit.hash}))`
}

function renderSections(items) {
  const sections = { features: [], improvements: [], fixes: [], breaking: [], technical: [] }
  for (const item of items) {
    sections[item.section].push(item)
    if (item.breaking) sections.breaking.push(item)
  }
  return sections
}

const date = new Date().toISOString().slice(0, 10)
const prev = previousTag()
const range = prev ? `${prev}..HEAD` : "HEAD"
const items = commits(range).map(classify).filter(Boolean)
const sections = renderSections(items)

const lines = [
  `# Postwoman v${version}`,
  "",
  `**Release date:** ${date}`,
  "",
  `Postwoman v${version} is an API development desktop release for Windows and Linux.`,
  "",
]

function pushSection(title, key) {
  const list = sections[key]
  if (!list.length) return
  lines.push(`## ${title}`, "")
  const limit = key === "technical" ? 12 : list.length
  for (const item of list.slice(0, limit)) lines.push(`- ${bullet(item)}`)
  if (list.length > limit) lines.push(`- …and ${list.length - limit} additional maintenance commits`)
  lines.push("")
}

if (items.length === 0) {
  lines.push("## Summary", "", "This is a maintenance release with no user-facing feature or fix commits since the previous release.", "")
} else {
  pushSection("Features", "features")
  pushSection("Improvements", "improvements")
  pushSection("Bug Fixes", "fixes")
  pushSection("Breaking Changes", "breaking")
  pushSection("Technical Changes", "technical")
}

if (prev) {
  lines.push(`**Full changelog:** ${repoURL}/compare/${prev}...v${version}`, "")
}

lines.push(
  "## Downloads",
  "",
  "| Platform | Asset | Notes |",
  "| --- | --- | --- |",
  `| Windows (installer) | \`Postwoman-v${version}-windows-x64-setup.exe\` | Recommended. Per-user install, no administrator privileges, Start Menu shortcut, optional desktop shortcut, clean upgrades. |`,
  `| Windows (portable) | \`Postwoman-v${version}-windows-x64-portable.zip\` | Extract anywhere and run \`Postwoman.exe\`. Data stays in the adjacent \`data\` folder. |`,
  `| Debian / Ubuntu | \`postwoman_${version}_amd64.deb\` | Native package with GTK/WebKitGTK dependencies resolved by apt. |`,
  `| Linux (portable) | \`Postwoman-${version}-x86_64.AppImage\` | Single portable executable; no installation required. |`,
  "| Checksums | `SHA256SUMS.txt` | SHA-256 hashes for every attached asset. |",
  "",
  "## Installation",
  "",
  "### Windows — installer (recommended)",
  "",
  `Download \`Postwoman-v${version}-windows-x64-setup.exe\` and run it. The installer is per-user, installs under \`%LOCALAPPDATA%\\Programs\\Postwoman\`, creates a Start Menu shortcut and lets you choose whether to add a desktop shortcut. It installs the Microsoft WebView2 Runtime when missing. Upgrades overwrite the previous installation; your data in \`%APPDATA%\\Postwoman\` is preserved.`,
  "",
  "### Windows — portable",
  "",
  `Download \`Postwoman-v${version}-windows-x64-portable.zip\`, extract the whole folder to a writable location (USB drive included) and run \`Postwoman.exe\`. The included \`portable.txt\` marker keeps the database and WebView2 cache in the folder's \`data\` directory. Keep \`Postwoman.exe\`, \`portable.txt\` and \`data\` together when moving the app.`,
  "",
  "### Linux",
  "",
  `Install the Debian package with \`sudo apt install ./postwoman_${version}_amd64.deb\`, or make the AppImage executable with \`chmod +x Postwoman-${version}-x86_64.AppImage\` and run it.`,
  "",
  "## Verifying downloads",
  "",
  "Download `SHA256SUMS.txt` alongside the assets and verify a file with:",
  "",
  "```bash",
  "sha256sum -c SHA256SUMS.txt",
  "```",
  "",
  "On Windows PowerShell:",
  "",
  "```powershell",
  "Get-FileHash .\\Postwoman-v" + version + "-windows-x64-setup.exe -Algorithm SHA256",
  "```",
  "",
  "Requires Windows 10/11 x64 or a 64-bit Linux distribution with GTK 3 and WebKitGTK 4.1.",
  "",
)

function changelogEntry() {
  const out = [`## [${version}] - ${date}`, ""]
  const section = (title, key) => {
    if (!sections[key].length) return
    out.push(`### ${title}`, "")
    for (const item of sections[key]) out.push(`- ${bullet(item)}`)
    out.push("")
  }
  section("Features", "features")
  section("Improvements", "improvements")
  section("Bug Fixes", "fixes")
  section("Breaking Changes", "breaking")
  section("Technical Changes", "technical")
  if (Object.values(sections).every((list) => list.length === 0)) {
    out.push("Maintenance release with no user-facing changes.", "")
  }
  out.push(`[${version}]: ${repoURL}/releases/tag/v${version}`, "")
  return out.join("\n")
}

const notes = `${lines.join("\n").replace(/\n{3,}/g, "\n\n")}\n`
const outPath = path.resolve(root, options.out || `docs/RELEASE_NOTES_v${version}.md`)
if (options.stdout) {
  process.stdout.write(notes)
} else {
  fs.mkdirSync(path.dirname(outPath), { recursive: true })
  fs.writeFileSync(outPath, notes)
  console.log(`wrote ${path.relative(root, outPath)}`)
}

if (!options.noChangelog) {
  const changelogPath = path.resolve(root, options.changelog)
  let changelog = fs.existsSync(changelogPath)
    ? fs.readFileSync(changelogPath, "utf8")
    : "# Changelog\n\nAll notable changes to Postwoman are documented here. The format follows Keep a Changelog and Semantic Versioning.\n"
  const entry = changelogEntry()
  const heading = `## [${version}]`
  const start = changelog.indexOf(heading)
  if (start >= 0) {
    const next = changelog.indexOf("\n## [", start + heading.length)
    changelog = changelog.slice(0, start) + entry + (next >= 0 ? changelog.slice(next + 1) : "\n")
  } else {
    const first = changelog.indexOf("\n## [")
    if (first >= 0) {
      changelog = `${changelog.slice(0, first + 1)}${entry}\n${changelog.slice(first + 1)}`
    } else {
      changelog = `${changelog.replace(/\s*$/, "")}\n\n${entry}\n`
    }
  }
  fs.writeFileSync(changelogPath, changelog)
  console.log(`updated ${path.relative(root, changelogPath)}`)
}

console.log(`release notes for v${version}: ${items.length} commit(s), previous tag: ${prev || "(none)"}`)