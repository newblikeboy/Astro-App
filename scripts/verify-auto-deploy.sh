#!/usr/bin/env bash
# Exercise deployment and rollback in disposable Git repositories; no real services, builds or credentials are used.
set -euo pipefail
project_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
mkdir -p "$project_dir/.local"
qa_dir=$(mktemp -d "$project_dir/.local/deploy-qa.XXXXXXXX")
real_git=$(command -v git)

# cleanup removes only the verified disposable workspace created by this test run.
cleanup() {
  case "$qa_dir" in "$project_dir"/.local/deploy-qa.*) rm -rf -- "$qa_dir" ;; *) exit 1 ;; esac
}
trap cleanup EXIT
mkdir -p "$qa_dir"/{mock-bin,state,releases,cache,publisher}
git init --quiet --bare "$qa_dir/origin.git"
git -C "$qa_dir/publisher" init --quiet --initial-branch=main
git -C "$qa_dir/publisher" config user.name 'Deployment QA'
git -C "$qa_dir/publisher" config user.email 'deployment-qa@example.invalid'
git -C "$qa_dir/publisher" config core.autocrlf false
printf 'bin/\nweb/dist/\n' > "$qa_dir/publisher/.gitignore"
printf 'initial source\n' > "$qa_dir/publisher/content.txt"
git -C "$qa_dir/publisher" add .
git -C "$qa_dir/publisher" commit --quiet -m initial
git -C "$qa_dir/publisher" remote add origin "$qa_dir/origin.git"
git -C "$qa_dir/publisher" push --quiet origin main
git clone --quiet --branch main "$qa_dir/origin.git" "$qa_dir/app"
git -C "$qa_dir/app" config core.autocrlf false
mkdir -p "$qa_dir/app/bin" "$qa_dir/app/web/dist"
printf 'original binary\n' > "$qa_dir/app/bin/mishra4you"
printf 'original frontend\n' > "$qa_dir/app/web/dist/index.html"
ln -s "$qa_dir/app" "$qa_dir/current"
[[ -L "$qa_dir/current" ]] || { printf 'Bash-compatible symlinks are required (Git Bash: MSYS=winsymlinks:sys).\n' >&2; exit 1; }

# mock_header starts each tool shim with its isolated test paths rather than production configuration.
mock_header() {
  printf '#!/usr/bin/env bash\nset -euo pipefail\nqa_dir=%q\nreal_git=%q\n' "$qa_dir" "$real_git" > "$qa_dir/mock-bin/$1"
}
mock_header git
cat >> "$qa_dir/mock-bin/git" <<'MOCK'
# MSYS Git reports drive paths while Bash readlink reports /d/... paths.
if [[ "$*" == *'rev-parse --show-toplevel'* ]] && command -v cygpath >/dev/null; then
  cygpath -u "$("$real_git" "$@")"
else
  "$real_git" "$@"
fi
MOCK
mock_header chown
printf 'exit 0\n' >> "$qa_dir/mock-bin/chown"
mock_header flock
printf '[[ ! -f "$qa_dir/locked" ]]\n' >> "$qa_dir/mock-bin/flock"
mock_header runuser
printf 'shift 3\nexec "$@"\n' >> "$qa_dir/mock-bin/runuser"
mock_header systemctl
cat >> "$qa_dir/mock-bin/systemctl" <<'MOCK'
printf '%s\n' "$*" >> "$qa_dir/restarts"
if [[ -f "$qa_dir/restart-fail" ]]; then rm "$qa_dir/restart-fail"; exit 1; fi
MOCK
mock_header curl
cat >> "$qa_dir/mock-bin/curl" <<'MOCK'
if [[ -f "$qa_dir/current/HEALTH_FAIL" ]]; then printf '{"status":"bad"}'; else printf '{"status":"ok"}'; fi
MOCK
mock_header go
cat >> "$qa_dir/mock-bin/go" <<'MOCK'
[[ "$ENV_FILE" == none && -n "$SOURCE_REVISION" && -z "${QA_PRIVATE_SECRET:-}" ]]
printf 'go %s\n' "$1" >> "$qa_dir/builds"
if [[ "$1" == test ]]; then [[ ! -f BUILD_FAIL ]]; exit; fi
mkdir -p bin
cat > bin/mishra4you <<'BIN'
#!/usr/bin/env bash
printf '%s\n' "$EPHE_DIR" "$FONT_PATH" "${QA_PRIVATE_SECRET:-}" "${HOME:-}"
BIN
chmod +x bin/mishra4you
MOCK
mock_header npm
cat >> "$qa_dir/mock-bin/npm" <<'MOCK'
printf 'npm %s\n' "$*" >> "$qa_dir/builds"
if [[ "$*" == *'run build'* ]]; then mkdir -p web/dist; printf '%s\n' "$SOURCE_REVISION" > web/dist/index.html; fi
MOCK
chmod +x "$qa_dir/mock-bin/"*
{
  printf 'APP_DIR=%q\nSTATE_DIR=%q\nRELEASE_ROOT=%q\nCURRENT_LINK=%q\nBUILD_CACHE=%q\n' "$qa_dir/app" "$qa_dir/state" "$qa_dir/releases" "$qa_dir/current" "$qa_dir/cache"
  printf 'BUILD_SCRIPT=%q\nBUILD_PATH=%q\nBUILD_USER=qa-builder\n' "$project_dir/deploy/build-release.sh" "$qa_dir/mock-bin:$PATH"
} > "$qa_dir/config"
export M4Y_DEPLOY_CONFIG="$qa_dir/config" QA_PRIVATE_SECRET=must-not-reach-builds
checks=0

# poll runs the actual orchestrator and requires its exit status to match the expected scenario.
poll() {
  local expected=$1 actual=0
  bash "$project_dir/deploy/auto-deploy.sh" > "$qa_dir/last-poll.log" 2>&1 || actual=$?
  if [[ "$expected" == success && "$actual" -ne 0 || "$expected" == failure && "$actual" -eq 0 ]]; then
    cat "$qa_dir/last-poll.log"; printf 'Unexpected deployment exit status: %s\n' "$actual" >&2; exit 1
  fi
}

# passed records a completed behavior assertion with a short readable result.
passed() { checks=$((checks+1)); printf 'PASS %s\n' "$*"; }

# publish commits the current fixture changes and advances the disposable remote main branch.
publish() {
  git -C "$qa_dir/publisher" add -A
  git -C "$qa_dir/publisher" commit --quiet -m "$1"
  git -C "$qa_dir/publisher" push --quiet origin main
  published=$(git -C "$qa_dir/publisher" rev-parse HEAD)
}

poll success
initial=$(git -C "$qa_dir/publisher" rev-parse HEAD)
[[ "$(cat "$qa_dir/state/revision")" == "$initial" && "$(cat "$qa_dir/current/web/dist/index.html")" == "$initial" ]]
[[ "$(cat "$qa_dir/builds")" == $'go test\ngo build\nnpm --prefix web ci --include=dev\nnpm --prefix web run build' ]]
[[ "$(git -C "$qa_dir/app" rev-parse HEAD)" == "$initial" ]]
good_target=$(readlink -f "$qa_dir/current")
passed 'first deployment tests/builds the published commit and promotes matching frontend/backend'
before=$(wc -l < "$qa_dir/restarts")
builds_before=$(wc -l < "$qa_dir/builds")
poll success
[[ $(wc -l < "$qa_dir/restarts") -eq "$before" && $(wc -l < "$qa_dir/builds") -eq "$builds_before" ]]
passed 'unchanged main does not build or restart'

touch "$qa_dir/publisher/BUILD_FAIL"
publish broken-build
poll failure
[[ "$(readlink -f "$qa_dir/current")" == "$good_target" && "$(cat "$qa_dir/state/revision")" == "$initial" && $(wc -l < "$qa_dir/restarts") -eq "$before" ]]
[[ -z "$(find "$qa_dir/cache" -maxdepth 1 -name 'work.*' -print)" ]]
passed 'failed tests leave the running release untouched and clean temporary files'
rm "$qa_dir/publisher/BUILD_FAIL"
touch "$qa_dir/publisher/HEALTH_FAIL"
publish unhealthy-release
poll failure
[[ "$(readlink -f "$qa_dir/current")" == "$good_target" && "$(cat "$qa_dir/state/revision")" == "$initial" && ! -e "$qa_dir/state/pending" ]]
[[ $(wc -l < "$qa_dir/restarts") -eq $((before+2)) ]]
passed 'unhealthy promotion rolls back the pointer, service and revision'
rm "$qa_dir/publisher/HEALTH_FAIL"
printf 'updated source\n' > "$qa_dir/publisher/content.txt"
publish healthy-release
touch "$qa_dir/restart-fail"
poll failure
[[ "$(readlink -f "$qa_dir/current")" == "$good_target" && ! -e "$qa_dir/state/pending" ]]
passed 'failed service restart rolls back'
poll success
good_revision=$published
good_target=$(readlink -f "$qa_dir/current")
[[ "$(cat "$qa_dir/state/revision")" == "$good_revision" && "$(cat "$qa_dir/current/content.txt")" == 'updated source' ]]
[[ "$(git -C "$qa_dir/app" rev-parse HEAD)" == "$initial" ]]
passed 'later healthy pushes deploy without resetting the original checkout'

printf 'operator edit\n' > "$qa_dir/app/content.txt"
poll failure
[[ "$(cat "$qa_dir/app/content.txt")" == 'operator edit' && "$(readlink -f "$qa_dir/current")" == "$good_target" ]]
git -C "$qa_dir/app" restore content.txt
passed 'dirty checkout is preserved and blocks deployment'
touch "$qa_dir/locked"
poll success
rm "$qa_dir/locked"
[[ "$(readlink -f "$qa_dir/current")" == "$good_target" ]]
passed 'held deployment lock skips the check'
git --git-dir="$qa_dir/origin.git" update-ref refs/heads/main "$initial"
poll failure
[[ "$(cat "$qa_dir/state/revision")" == "$good_revision" ]]
git --git-dir="$qa_dir/origin.git" update-ref refs/heads/main "$good_revision"
passed 'backward remote history cannot replace a newer deployed revision'

failed_target="$qa_dir/releases/$initial-1-1"
mkdir "$failed_target"
touch "$failed_target/HEALTH_FAIL"
printf '%s\n%s\n' "$good_target" "$good_revision" > "$qa_dir/state/pending"
rm "$qa_dir/current"
ln -s "$failed_target" "$qa_dir/current"
poll success
[[ "$(readlink -f "$qa_dir/current")" == "$good_target" && ! -e "$qa_dir/state/pending" && "$(cat "$qa_dir/state/revision")" == "$good_revision" ]]
passed 'pending interrupted promotion recovers before checking for new commits'

launcher_output=$(cd "$good_target" && EPHE_DIR="$qa_dir/app/ephe" FONT_PATH="$qa_dir/app/assets/NotoSansDevanagari-Regular.ttf" bash "$project_dir/deploy/run-current.sh")
[[ "$launcher_output" == "$good_target/ephe"$'\n'"$good_target/assets/NotoSansDevanagari-Regular.ttf"$'\n'"$QA_PRIVATE_SECRET"$'\n'"$HOME" ]]
launcher_output=$(cd "$good_target" && EPHE_DIR=/custom/ephe FONT_PATH=/custom/font.ttf bash "$project_dir/deploy/run-current.sh")
[[ "$launcher_output" == /custom/ephe$'\n'/custom/font.ttf$'\n'"$QA_PRIVATE_SECRET"$'\n'"$HOME" ]]
passed 'launcher maps release assets while preserving custom paths and app environment'
printf '\n%s deployment behavior checks passed.\n' "$checks"
