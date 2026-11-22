#!/usr/bin/env bash
#
# Prepare, commit and tag a Postwoman release.
#
# Usage:
#   ./scripts/release.sh 1.4.0
#   ./scripts/release.sh 1.4.0 --no-push
#   ./scripts/release.sh 1.4.0 --dry-run
#
# The script:
#   1. validates the semantic version and a clean working tree;
#   2. updates every version-bearing file from VERSION;
#   3. generates docs/RELEASE_NOTES_v<version>.md and CHANGELOG.md;
#   4. commits the release, creates an annotated v<version> tag;
#   5. pushes the branch and tag, which triggers the GitHub Actions release.
#
# Running this command is an explicit request to publish. Use --no-push to
# prepare the commit/tag locally, or --dry-run to inspect the changes only.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION=""
PUSH=1
DRY_RUN=0
ALLOW_DIRTY=0

for arg in "$@"; do
  case "$arg" in
    --no-push) PUSH=0 ;;
    --dry-run) DRY_RUN=1 ;;
    --allow-dirty) ALLOW_DIRTY=1 ;;
    -h|--help)
      sed -n '2,24p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    -*)
      echo "error: unknown option $arg" >&2
      exit 1
      ;;
    *)
      if [ -n "$VERSION" ]; then
        echo "error: only one version may be supplied" >&2
        exit 1
      fi
      VERSION="$arg"
      ;;
  esac
done

if ! [[ "$VERSION" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "error: version must be MAJOR.MINOR.PATCH (for example 1.4.0), got '${VERSION:-}'" >&2
  exit 1
fi

for tool in git node; do
  command -v "$tool" >/dev/null 2>&1 || { echo "error: $tool is required" >&2; exit 1; }
done

TAG="v$VERSION"
if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
  echo "error: tag $TAG already exists" >&2
  exit 1
fi

if [ "$PUSH" -eq 1 ] && ! git remote get-url origin >/dev/null 2>&1; then
  echo "error: no 'origin' remote configured (use --no-push to prepare locally)" >&2
  exit 1
fi
if [ "$PUSH" -eq 1 ] && git ls-remote --tags origin "refs/tags/$TAG" | grep -q "$TAG"; then
  echo "error: tag $TAG already exists on origin" >&2
  exit 1
fi

if [ "$ALLOW_DIRTY" -eq 0 ] && { ! git diff --quiet || ! git diff --cached --quiet; }; then
  echo "error: working tree has uncommitted changes; commit or stash them first" >&2
  echo "       (use --allow-dirty if you intentionally want to include them)" >&2
  exit 1
fi

echo "==> updating version metadata to $VERSION"
node scripts/update-version.mjs "$VERSION"

echo "==> generating release notes and changelog"
node scripts/generate-release-notes.mjs "$VERSION"

echo "==> verifying version consistency"
node scripts/update-version.mjs --check "$VERSION"

FILES=(
  VERSION
  app.go
  frontend/package.json
  frontend/package-lock.json
  wails.json
  README.md
  CHANGELOG.md
  "docs/RELEASE_NOTES_${TAG}.md"
)

if [ "$DRY_RUN" -eq 1 ]; then
  echo
  echo "Dry run complete. Release files that would be committed:"
  git diff --stat -- "${FILES[@]}"
  git status --short -- "${FILES[@]}"
  echo
  echo "Review docs/RELEASE_NOTES_${TAG}.md, then run: ./scripts/release.sh $VERSION"
  exit 0
fi

git add "${FILES[@]}"
if git diff --cached --quiet; then
  echo "error: no release changes were staged; is $VERSION already current?" >&2
  exit 1
fi

git commit -m "chore(release): $TAG"
git tag -a "$TAG" -m "Postwoman $TAG"

echo "==> created commit and tag $TAG"

if [ "$PUSH" -eq 0 ]; then
  echo
  echo "Nothing was pushed. When ready run:"
  echo "  git push origin $(git rev-parse --abbrev-ref HEAD)"
  echo "  git push origin $TAG"
  exit 0
fi

BRANCH="$(git rev-parse --abbrev-ref HEAD)"
echo "==> pushing $BRANCH and $TAG"
git push origin "$BRANCH"
git push origin "$TAG"

REMOTE_URL="$(git remote get-url origin)"
REPO_PATH="$REMOTE_URL"
case "$REPO_PATH" in
  *github.com[:/]*) REPO_PATH="${REPO_PATH#*github.com}" ; REPO_PATH="${REPO_PATH#[:/]}" ;;
esac
REPO_PATH="${REPO_PATH%.git}"
echo
echo "Release workflow triggered for $TAG."
if [[ "$REMOTE_URL" == *github.com* ]] && [[ "$REPO_PATH" == */* ]]; then
  echo "Actions: https://github.com/$REPO_PATH/actions/workflows/release.yml"
  echo "Release: https://github.com/$REPO_PATH/releases/tag/$TAG"
fi