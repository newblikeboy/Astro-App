#!/usr/bin/env python3
"""Read-only droplet checks. Output contains flags, never credentials or raw config."""

import argparse
import ipaddress
import json
from pathlib import Path
import re
import shlex
import stat
import subprocess
from urllib.parse import parse_qs, urlsplit


def read_environment(path):
    """Parse the simple systemd environment file without sourcing executable shell code."""
    values = {}
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if not line or line.startswith("#"):
            continue
        key, separator, raw = line.partition("=")
        if not separator or not re.fullmatch(r"[A-Z][A-Z0-9_]*", key):
            raise ValueError("unsupported environment syntax")
        words = shlex.split(raw, comments=False)
        values[key] = " ".join(words)
    return values


def configured(value):
    """Recognize blank and published template credentials without returning their values."""
    upper = value.upper()
    return bool(value.strip()) and not any(marker in upper for marker in ("REPLACE_", "CHANGE-ME", "CHANGE_ME"))


def configuration_checks(values):
    """Return only configuration booleans, including unsafe Mongo TLS overrides."""
    uri = urlsplit(values.get("MONGODB_URI", ""))
    options = {key.lower(): [v.lower() for v in vals] for key, vals in parse_qs(uri.query).items()}
    insecure_tls = any("true" in options.get(key, []) for key in ("tlsinsecure", "tlsallowinvalidcertificates", "tlsallowinvalidhostnames"))
    insecure_tls |= any("false" in options.get(key, []) for key in ("tls", "ssl"))
    address = values.get("HTTP_ADDR", "")
    try:
        loopback = ipaddress.ip_address(address.rsplit(":", 1)[0].strip("[]")).is_loopback
    except ValueError:
        loopback = False
    return {
        "production_mode": values.get("APP_ENV") == "production",
        "https_origin": values.get("APP_ORIGIN", "").startswith("https://"),
        "backend_bound_to_loopback": loopback,
        "loopback_proxy_trust_enabled": values.get("TRUST_LOOPBACK_PROXY") == "true",
        "admin_password_not_template": configured(values.get("ADMIN_PASSWORD", "")),
        "admin_password_at_least_16_characters": len(values.get("ADMIN_PASSWORD", "")) >= 16,
        "admin_email_not_example": values.get("ADMIN_EMAIL", "").lower() not in ("", "admin@example.com", "local-admin@example.test"),
        "mongo_credentials_not_template": configured(values.get("MONGODB_URI", "")),
        "mongo_tls_enabled_without_insecure_overrides": (uri.scheme == "mongodb+srv" or "true" in options.get("tls", []) or "true" in options.get("ssl", [])) and not insecure_tls,
        "payment_credentials_configured": all(configured(values.get(k, "")) for k in ("RAZORPAY_KEY_ID", "RAZORPAY_KEY_SECRET", "RAZORPAY_WEBHOOK_SECRET")),
        "payment_webhook_secret_configured": configured(values.get("RAZORPAY_WEBHOOK_SECRET", "")),
        "meeting_credentials_configured": all(configured(values.get(k, "")) for k in ("ZOOM_ACCOUNT_ID", "ZOOM_CLIENT_ID", "ZOOM_CLIENT_SECRET")),
        "notification_credentials_configured": all(configured(values.get(k, "")) for k in ("NOTIFICATION_URL", "NOTIFICATION_TOKEN")),
        "simulations_disabled": values.get("PAYMENT_MODE") == "razorpay" and values.get("ZOOM_MODE") == "zoom",
    }


def capture(arguments):
    """Capture bounded diagnostic commands privately; callers expose only selected safe fields."""
    try:
        result = subprocess.run(arguments, capture_output=True, text=True, timeout=20, check=False)
        return result.returncode == 0, result.stdout
    except (OSError, subprocess.TimeoutExpired):
        return False, ""


def main():
    """Print sanitized checks without modifying services, files, accounts or customer data."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--env-file", type=Path, default=Path("/etc/mishra4you/app.env"))
    args = parser.parse_args()
    report = {"read_only": True, "secrets_included": False}
    try:
        metadata = args.env_file.stat()
        parent = args.env_file.parent.stat()
        report["environment_file_private"] = metadata.st_uid == 0 and not stat.S_IMODE(metadata.st_mode) & 0o077
        report["environment_parent_not_publicly_writable"] = parent.st_uid == 0 and not stat.S_IMODE(parent.st_mode) & 0o022
        report["configuration"] = configuration_checks(read_environment(args.env_file))
    except (OSError, ValueError):
        report["environment_readable_and_parseable"] = False
    fields = ("User", "Group", "ProtectSystem", "ProtectHome", "PrivateTmp", "UMask", "ReadWritePaths", "ActiveState")
    ok, output = capture(["systemctl", "show", "mishra4you.service", "--property=" + ",".join(fields)])
    report["service_inspected"] = ok
    if ok:
        service = dict(line.split("=", 1) for line in output.splitlines() if "=" in line)
        report["service"] = {"non_root_user": service.get("User") not in (None, "", "root"), "protect_system_strict": service.get("ProtectSystem") == "strict", "protect_home": service.get("ProtectHome") == "yes", "private_tmp": service.get("PrivateTmp") == "yes", "private_file_umask": service.get("UMask") == "0077", "active": service.get("ActiveState") == "active"}
    ok, output = capture(["nginx", "-T"])
    report["nginx_configuration_valid"] = ok
    # Presence alone does not prove every active location is configured; require manual scope review.
    report["nginx_overwrite_directive_present"] = bool(ok and re.search(r"^\s*proxy_set_header\s+X-Real-IP\s+\$remote_addr\s*;", output, re.M))
    ok, output = capture(["ufw", "status"])
    report["ufw_active"] = bool(ok and re.search(r"Status:\s+active", output))
    ok, output = capture(["sshd", "-T"])
    report["ssh_inspected"] = ok
    if ok:
        ssh = dict(line.split(" ", 1) for line in output.splitlines() if " " in line)
        report["ssh_password_auth_disabled"] = ssh.get("passwordauthentication") == "no"
        report["ssh_root_password_auth_disabled"] = ssh.get("permitrootlogin") in ("no", "without-password", "prohibit-password")
    report["requires_separate_verification"] = ["Atlas IP allowlist and database-user privileges", "off-host backups and restore test", "Ubuntu and Chrome security updates", "all public listeners and cloud firewall", "effective Nginx proxy locations", "GitHub account protection and branch access", "existing access logs for evidence of compromise"]
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
