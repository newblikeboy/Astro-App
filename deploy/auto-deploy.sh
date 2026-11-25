#!/usr/bin/env bash
# Poll main, build an isolated release, then switch the running service with health-checked rollback.
set -Eeuo pipefail
config=${M4Y_DEPLOY_CONFIG:-/etc/default/mishra4you-deploy}
if [[ -r "$config" ]]; then source "$config"; fi
APP_DIR=${APP_DIR:-/opt/mishra4you}
BRANCH=${BRANCH:-main}
APP_SERVICE=${APP_SERVICE:-mishra4you.service}
STATE_DIR=${STATE_DIR:-/var/lib/mishra4you-deploy}
RELEASE_ROOT=${RELEASE_ROOT:-/opt/mishra4you-releases}
CURRENT_LINK=${CURRENT_LINK:-/opt/mishra4you-current}
BUILD_USER=${BUILD_USER:-mishra4you-build}
BUILD_CACHE=${BUILD_CACHE:-/var/cache/mishra4you-build}
BUILD_SCRIPT=${BUILD_SCRIPT:-/usr/local/lib/mishra4you/build-release.sh}
BUILD_PATH=${BUILD_PATH:-/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin}
HEALTH_URL=${HEALTH_URL:-http://127.0.0.1:8080/api/health}
# Administrative commands such as Ubuntu's runuser live outside the build tool PATH.
export PATH="$BUILD_PATH:/usr/local/sbin:/usr/sbin:/sbin"
export GIT_TERMINAL_PROMPT=0
workspace= candidate= link_temp=
promoted=0

# log records deployment decisions in the systemd journal without displaying configuration values.
log() { printf '[mishra4you-deploy] %s\n' "$*"; }

# valid_target confines all live/rollback pointers to the original checkout or a managed release directory.
valid_target() {
  local name=${1##*/}
  [[ -d "$1" && ! -L "$1" ]] && { [[ "$1" == "$APP_DIR" ]] || { [[ "${1%/*}" == "$RELEASE_ROOT" ]] && [[ "$name" =~ ^[a-f0-9]{40}-[0-9]+-[0-9]+$ ]]; }; }
}

# switch_release atomically replaces the symlink used by the app's working directory and launcher.
switch_release() {
  valid_target "$1" || { log 'Refusing an invalid release target.'; return 1; }
  link_temp="${CURRENT_LINK}.next.$$"
  ln -s -- "$1" "$link_temp" || return 1
  mv -Tf -- "$link_temp" "$CURRENT_LINK" || return 1
  link_temp=
}

# healthy requires both a successful HTTP response and the app's documented healthy status.
healthy() {
  local body
  body=$(curl --fail --silent --show-error --retry 15 --retry-connrefused --retry-delay 2 --max-time 5 "$HEALTH_URL") || return 1
  [[ "$body" =~ \"status\"[[:space:]]*:[[:space:]]*\"ok\" ]]
}

# rollback_pending restores the previous pointer and revision, including recovery after an interrupted deployment.
rollback_pending() {
  local old_target old_revision
  { IFS= read -r old_target; IFS= read -r old_revision || true; } < "$STATE_DIR/pending"
  valid_target "$old_target" || return 1
  [[ -z "$old_revision" || "$old_revision" =~ ^[a-f0-9]{40}$ ]] || return 1
  switch_release "$old_target" || return 1
  systemctl restart "$APP_SERVICE" || return 1
  healthy || return 1
  if [[ -n "$old_revision" ]]; then
    printf '%s\n' "$old_revision" > "$STATE_DIR/revision.next"
    mv -f -- "$STATE_DIR/revision.next" "$STATE_DIR/revision"
  else
    rm -f -- "$STATE_DIR/revision"
  fi
  rm -f -- "$STATE_DIR/pending"
  log 'Previous release restored and healthy.'
}

# finish rolls back any incomplete promotion and deletes only this run's managed temporary files.
finish() {
  local status=$?
  trap - EXIT HUP INT TERM
  if (( status != 0 && promoted )); then
    log 'Deployment failed; restoring the previous release.'
    rollback_pending || log 'Rollback failed; the pending marker is retained for recovery on the next check.'
  fi
  if [[ -n "$link_temp" && "$link_temp" == "${CURRENT_LINK}.next."* ]]; then rm -f -- "$link_temp"; fi
  if [[ -n "$workspace" && "$workspace" == "$BUILD_CACHE/work."* ]]; then rm -rf -- "$workspace"; fi
  if (( status != 0 )) && [[ -n "$candidate" && "$candidate" == "$RELEASE_ROOT/"* && ! -L "$candidate" && "$(readlink -f -- "$CURRENT_LINK")" != "$candidate" ]]; then
    rm -rf -- "$candidate"
  fi
  exit "$status"
}
trap finish EXIT
trap 'exit 143' HUP INT TERM

# prune_releases retains the three newest managed releases plus any current or rollback target.
prune_releases() {
  local entry name count=0
  while IFS= read -r -d '' entry; do
    entry=${entry#* }
    name=${entry##*/}
    [[ "$name" =~ ^[a-f0-9]{40}-[0-9]+-[0-9]+$ ]] || continue
    count=$((count+1))
    if (( count > 3 )) && [[ "$entry" != "$candidate" && "$entry" != "$previous_target" && ! -L "$entry" ]]; then
      rm -rf -- "$entry" || log 'Could not remove an older release.'
    fi
  done < <(find "$RELEASE_ROOT" -mindepth 1 -maxdepth 1 -type d -printf '%T@ %p\0' | sort -zrn)
}

for tool in git tar chown runuser systemctl curl flock; do
  command -v "$tool" >/dev/null || { log "Missing deployment tool: $tool. Check the system PATH and installed packages."; exit 1; }
done
[[ -d "$STATE_DIR" && -d "$RELEASE_ROOT" && -d "$BUILD_CACHE" && -L "$CURRENT_LINK" ]] || { log 'Run the one-time installer first.'; exit 1; }
STATE_DIR=$(readlink -f -- "$STATE_DIR")
RELEASE_ROOT=$(readlink -f -- "$RELEASE_ROOT")
BUILD_CACHE=$(readlink -f -- "$BUILD_CACHE")
exec 9> "$STATE_DIR/deploy.lock"
flock -n 9 || { log 'Another deployment is already running.'; exit 0; }
APP_DIR=$(readlink -f -- "$APP_DIR")
[[ "$(git -C "$APP_DIR" rev-parse --show-toplevel)" == "$APP_DIR" ]] || { log 'The app folder must be the repository root.'; exit 1; }
git check-ref-format "refs/heads/$BRANCH"
if [[ -f "$STATE_DIR/pending" ]]; then
  log 'Recovering an interrupted deployment.'
  rollback_pending
fi
[[ -z "$(git -C "$APP_DIR" status --porcelain)" ]] || { log 'Checkout has local changes; leaving it and the running app untouched.'; exit 1; }
git -C "$APP_DIR" fetch --quiet --no-tags origin "refs/heads/$BRANCH:refs/remotes/origin/$BRANCH"
target=$(git -C "$APP_DIR" rev-parse --verify "refs/remotes/origin/$BRANCH^{commit}")
[[ "$target" =~ ^[a-f0-9]{40}$ ]] || exit 1
previous_revision=
if [[ -f "$STATE_DIR/revision" ]]; then IFS= read -r previous_revision < "$STATE_DIR/revision"; fi
[[ -z "$previous_revision" || "$previous_revision" =~ ^[a-f0-9]{40}$ ]] || { log 'Invalid recorded release revision.'; exit 1; }
previous_target=$(readlink -f -- "$CURRENT_LINK")
valid_target "$previous_target" || exit 1
if [[ "$target" == "$previous_revision" ]]; then log "No new commit on $BRANCH."; exit 0; fi
baseline=${previous_revision:-$(git -C "$APP_DIR" rev-parse HEAD)}
git -C "$APP_DIR" merge-base --is-ancestor "$baseline" "$target" || { log 'Remote history moved backwards or diverged; automatic deployment stopped.'; exit 1; }

log "Building published commit $target."
workspace=$(mktemp -d "$BUILD_CACHE/work.XXXXXXXX")
git -C "$APP_DIR" archive "$target" | tar -xf - -C "$workspace"
chown -R "$BUILD_USER:$BUILD_USER" "$workspace"
runuser -u "$BUILD_USER" -- env -i HOME="$BUILD_CACHE" PATH="$BUILD_PATH" SOURCE_REVISION="$target" bash "$BUILD_SCRIPT" "$workspace"
[[ -f "$workspace/bin/mishra4you" && ! -L "$workspace/bin/mishra4you" && -s "$workspace/web/dist/index.html" ]] || exit 1
[[ -z "$(find "$workspace/web/dist" -type l -print -quit)" ]] || { log 'Unexpected symlink in frontend output.'; exit 1; }

candidate="$RELEASE_ROOT/$target-$(date +%s)-$$"
mkdir -m 755 -- "$candidate"
git -C "$APP_DIR" archive "$target" | tar -xf - -C "$candidate"
install -d -m 755 "$candidate/bin" "$candidate/web"
install -m 755 "$workspace/bin/mishra4you" "$candidate/bin/mishra4you"
cp -R --no-dereference -- "$workspace/web/dist" "$candidate/web/dist"
chmod -R a+rX,go-w -- "$candidate"
[[ ! -f "$STATE_DIR/pending" ]] || exit 1
printf '%s\n%s\n' "$previous_target" "$previous_revision" > "$STATE_DIR/pending.next"
mv -f -- "$STATE_DIR/pending.next" "$STATE_DIR/pending"
promoted=1
switch_release "$candidate"
systemctl restart "$APP_SERVICE"
healthy
printf '%s\n' "$target" > "$STATE_DIR/revision.next"
mv -f -- "$STATE_DIR/revision.next" "$STATE_DIR/revision"
rm -f -- "$STATE_DIR/pending"
promoted=0
log "Deployment healthy: $target."
prune_releases
