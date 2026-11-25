#!/usr/bin/env bash
# Launch the selected release, remapping only the original checkout's asset paths.
set -euo pipefail
config=${M4Y_DEPLOY_CONFIG:-/etc/default/mishra4you-deploy}
if [[ -r "$config" ]]; then source "$config"; fi
APP_DIR=${APP_DIR:-/opt/mishra4you}
release_dir=$(pwd -P)
case "${EPHE_DIR:-ephe}" in
  "$APP_DIR/ephe"|ephe) export EPHE_DIR="$release_dir/ephe" ;;
esac
case "${FONT_PATH:-assets/NotoSansDevanagari-Regular.ttf}" in
  "$APP_DIR"/assets/*) export FONT_PATH="$release_dir/assets/${FONT_PATH##*/}" ;;
  assets/*) export FONT_PATH="$release_dir/${FONT_PATH:-assets/NotoSansDevanagari-Regular.ttf}" ;;
esac
exec "$release_dir/bin/mishra4you"
