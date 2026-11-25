#!/usr/bin/env bash
# Install polling deployment for the existing systemd app without replacing production secrets or private data.
set -euo pipefail
[[ $EUID -eq 0 ]] || { printf 'Run with sudo: sudo bash deploy/install-auto-deploy.sh\n' >&2; exit 1; }
export PATH="/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:$PATH"
source_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
app_dir=${APP_DIR:-$source_dir}
app_dir=$(readlink -f -- "$app_dir")
[[ "$app_dir" != *$'\n'* ]] || { printf 'App paths containing newlines are not supported.\n' >&2; exit 1; }
[[ "$(git -C "$app_dir" rev-parse --show-toplevel)" == "$app_dir" && -s "$app_dir/bin/mishra4you" && -s "$app_dir/web/dist/index.html" ]] || { printf 'An existing built deployment is required. See DEPLOYMENT.md first.\n' >&2; exit 1; }
for tool in git go gcc node npm curl tar flock runuser systemctl; do
  command -v "$tool" >/dev/null || { printf 'Missing required tool: %s\n' "$tool" >&2; exit 1; }
done
systemctl cat mishra4you.service >/dev/null
if [[ -r /etc/default/mishra4you-deploy ]]; then
  configured_app=$(bash -c 'source /etc/default/mishra4you-deploy; printf "%s" "${APP_DIR:-/opt/mishra4you}"')
  [[ "$configured_app" == "$app_dir" ]] || { printf 'Existing auto-deploy configuration points to another checkout.\n' >&2; exit 1; }
fi
workdir=$(systemctl show --property=WorkingDirectory --value mishra4you.service)
[[ "$workdir" == "$app_dir" || "$workdir" == /opt/mishra4you-current ]] || { printf 'The existing app service uses a different working directory.\n' >&2; exit 1; }

systemctl stop mishra4you-deploy.timer 2>/dev/null || true
systemctl stop mishra4you-deploy.service 2>/dev/null || true
if ! id mishra4you-build >/dev/null 2>&1; then
  useradd --system --user-group --home-dir /var/cache/mishra4you-build --shell /usr/sbin/nologin mishra4you-build
fi
install -d -o mishra4you-build -g mishra4you-build -m 750 /var/cache/mishra4you-build
install -d -m 700 /var/lib/mishra4you-deploy
install -d -m 755 /opt/mishra4you-releases /usr/local/lib/mishra4you /etc/systemd/system/mishra4you.service.d
if [[ -e /opt/mishra4you-current || -L /opt/mishra4you-current ]]; then
  [[ -L /opt/mishra4you-current ]] || { printf 'The current-release path is already occupied by a non-symlink.\n' >&2; exit 1; }
  current=$(readlink -f -- /opt/mishra4you-current)
  [[ -d "$current" && ! -L "$current" ]] || { printf 'The existing release pointer has no valid directory.\n' >&2; exit 1; }
  current_name=${current##*/}
  [[ "$current" == "$app_dir" || ( "${current%/*}" == /opt/mishra4you-releases && "$current_name" =~ ^[a-f0-9]{40}-[0-9]+-[0-9]+$ ) ]] || { printf 'Unexpected existing release pointer.\n' >&2; exit 1; }
else
  ln -s -- "$app_dir" /opt/mishra4you-current
fi
if [[ ! -e /etc/default/mishra4you-deploy ]]; then
  printf 'APP_DIR=%q\nBRANCH=main\n' "$app_dir" > /etc/default/mishra4you-deploy
  chmod 644 /etc/default/mishra4you-deploy
fi
install -m 755 "$source_dir/deploy/auto-deploy.sh" /usr/local/sbin/mishra4you-auto-deploy
install -m 755 "$source_dir/deploy/build-release.sh" /usr/local/lib/mishra4you/build-release.sh
install -m 755 "$source_dir/deploy/run-current.sh" /usr/local/lib/mishra4you/run-current.sh
install -m 644 "$source_dir/deploy/mishra4you-deploy.service" /etc/systemd/system/mishra4you-deploy.service
install -m 644 "$source_dir/deploy/mishra4you-deploy.timer" /etc/systemd/system/mishra4you-deploy.timer
cat > /etc/systemd/system/mishra4you.service.d/90-auto-deploy.conf <<'UNIT'
[Service]
WorkingDirectory=/opt/mishra4you-current
ExecStart=
ExecStart=/usr/local/lib/mishra4you/run-current.sh
UNIT
systemctl daemon-reload
systemctl show --property=ExecStart --value mishra4you.service | grep -Fq /usr/local/lib/mishra4you/run-current.sh
systemctl enable --now mishra4you-deploy.timer
systemctl start --no-block mishra4you-deploy.service
printf '\nAuto-deploy installed. Initial release build has been queued.\n'
printf 'Timer: systemctl list-timers mishra4you-deploy.timer\n'
printf 'Logs:  journalctl -u mishra4you-deploy.service -f\n'
