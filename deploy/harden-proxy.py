#!/usr/bin/env python3
"""Enable trusted client IPs for the documented local Nginx proxy; preserve TLS settings."""

import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import tempfile


def proxy_configuration(text):
    """Overwrite client IPs immediately beside each standalone local proxy directive; reject custom layouts."""
    target = re.compile(r"^(\s*)proxy_pass\s+http://127\.0\.0\.1:8080;\s*$")
    lines = text.splitlines()
    passes = [line for line in lines if re.search(r"\bproxy_pass\b", line) and not line.lstrip().startswith("#")]
    if not passes or any(not target.fullmatch(line) for line in passes):
        raise ValueError("unsupported proxy layout; review Nginx manually")
    for line in lines:
        if re.match(r"^\s*proxy_set_header\s+X-Real-IP\b", line, re.I) and not re.match(r"^\s*proxy_set_header\s+X-Real-IP\s+[^;]+;\s*$", line, re.I):
            raise ValueError("unsupported client IP directive")
    output = []
    for line in lines:
        if re.match(r"^\s*proxy_set_header\s+X-Real-IP\s+[^;]+;\s*$", line, re.I):
            continue
        output.append(line)
        match = target.fullmatch(line)
        if match:
            output.append(match.group(1) + "proxy_set_header X-Real-IP $remote_addr;")
    return "\n".join(output) + "\n"


def environment_configuration(text):
    """Set only the proxy trust flag while preserving every other credential line verbatim."""
    lines = [line for line in text.splitlines() if not re.match(r"^\s*TRUST_LOOPBACK_PROXY=", line)]
    return "\n".join(lines + ["TRUST_LOOPBACK_PROXY=true"]) + "\n"


def replace_file(path, content, metadata):
    """Replace one root-managed regular file atomically while preserving its ownership and mode."""
    descriptor, temporary = tempfile.mkstemp(prefix=".m4y-security-", dir=path.parent)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8", newline="\n") as output:
            output.write(content)
            output.flush()
            os.fsync(output.fileno())
        os.chown(temporary, metadata.st_uid, metadata.st_gid)
        os.chmod(temporary, metadata.st_mode & 0o777)
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def checked_command(arguments):
    """Run a bounded configuration check or reload without printing potentially private diagnostics."""
    result = subprocess.run(arguments, capture_output=True, timeout=30, check=False)
    if result.returncode:
        raise RuntimeError("configuration check or reload failed; diagnostics withheld")


def main():
    """Back up known files privately, validate and reload Nginx, then enable opt-in for the next app restart."""
    if not hasattr(os, "geteuid") or os.geteuid() != 0:
        raise ValueError("run with sudo on the documented Ubuntu droplet")
    nginx = Path("/etc/nginx/sites-available/mishra4you")
    environment = Path("/etc/mishra4you/app.env")
    for path in (nginx, environment):
        if path.is_symlink() or not path.is_file() or path.stat().st_uid != 0 or path.stat().st_mode & 0o022:
            raise ValueError("expected root-owned regular files without group/public write access")
    if environment.stat().st_mode & 0o077:
        raise ValueError("make /etc/mishra4you/app.env private with chmod 600 first")
    original_nginx = nginx.read_text(encoding="utf-8")
    original_environment = environment.read_text(encoding="utf-8")
    changed_nginx = proxy_configuration(original_nginx)
    changed_environment = environment_configuration(original_environment)
    checked_command(["nginx", "-t"])
    backup = Path(tempfile.mkdtemp(prefix="mishra4you-security-", dir="/root"))
    for path in (nginx, environment):
        shutil.copy2(path, backup / path.name)
    nginx_metadata, environment_metadata = nginx.stat(), environment.stat()
    try:
        replace_file(nginx, changed_nginx, nginx_metadata)
        checked_command(["nginx", "-t"])
        checked_command(["systemctl", "reload", "nginx"])
        replace_file(environment, changed_environment, environment_metadata)
    except (OSError, RuntimeError, subprocess.TimeoutExpired):
        replace_file(nginx, original_nginx, nginx_metadata)
        replace_file(environment, original_environment, environment_metadata)
        checked_command(["nginx", "-t"])
        checked_command(["systemctl", "reload", "nginx"])
        raise RuntimeError("proxy update failed and previous configuration was restored") from None
    print("Proxy headers validated. Trust flag prepared for the next app deployment.")
    print("Private configuration backup: " + str(backup))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.TimeoutExpired):
        # Errors never include configuration file contents or subprocess output.
        print("Proxy hardening did not complete; review the local configuration privately.", file=sys.stderr)
        sys.exit(1)
