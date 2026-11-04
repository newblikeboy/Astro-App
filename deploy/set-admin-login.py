#!/usr/bin/env python3
"""Configure administrator credentials privately in the server environment file."""

import argparse
import getpass
import os
from pathlib import Path
import re
import tempfile
from email.utils import parseaddr


def quote_environment(value):
    """Quote a value for systemd EnvironmentFile without expanding shell variables."""
    return '"' + value.replace('\\', '\\\\').replace('"', '\\"') + '"'


def save_credentials(path, email, password):
    """Atomically replace only admin credentials and remove the obsolete TOTP setting."""
    original = path.read_text(encoding="utf-8")
    lines = [line for line in original.splitlines()
             if not re.match(r"^(?:export\s+)?ADMIN_(?:EMAIL|PASSWORD|TOTP_SECRET)\s*=", line)]
    lines.extend([f"ADMIN_EMAIL={quote_environment(email)}",
                  f"ADMIN_PASSWORD={quote_environment(password)}"])
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8", newline="\n",
                                         dir=path.parent, delete=False) as stream:
            temporary = stream.name
            stream.write("\n".join(lines) + "\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(temporary, 0o600)
        os.replace(temporary, path)
        temporary = None
    finally:
        if temporary is not None:
            os.unlink(temporary)


def main():
    """Prompt for the password without printing it or accepting it in process arguments."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--email", required=True)
    parser.add_argument("--env-file", type=Path, default=Path("/etc/mishra4you/app.env"))
    args = parser.parse_args()
    email = args.email.strip().lower()
    name, parsed = parseaddr(email)
    if name or parsed != email or not re.fullmatch(r"[^\s@]+@[^\s@]+\.[^\s@]+", email):
        parser.error("Enter a valid administrator email address.")
    if not args.env_file.is_file():
        parser.error("The server environment file must already exist.")
    password = getpass.getpass("Administrator password: ")
    if len(password.encode("utf-8")) < 8 or password == "local-admin-change-me":
        parser.error("Use a private password of at least 8 characters.")
    if password != getpass.getpass("Confirm password: "):
        parser.error("Passwords do not match.")
    save_credentials(args.env_file, email, password)
    print("Administrator credentials saved privately. Restart mishra4you to apply them.")


if __name__ == "__main__":
    main()
