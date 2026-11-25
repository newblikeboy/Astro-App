"""Exercise deployment security tools with disposable configuration and mocked services."""

import importlib.util
import io
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch


def load_tool(name):
    """Load a deployment helper without executing its command-line entry point."""
    path = Path(__file__).resolve().parents[1] / "deploy" / (name + ".py")
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


check = load_tool("security-check")
proxy = load_tool("harden-proxy")


class SecurityToolTests(unittest.TestCase):
    """Verify redaction, conservative trust opt-in and rollback without touching a real server."""

    def test_read_only_report_never_prints_secrets(self):
        """Parse a private fixture and prove the report omits credentials and raw diagnostic output."""
        marker = "PRIVATE_FIXTURE_DO_NOT_PRINT_493810"
        with tempfile.TemporaryDirectory() as directory:
            env = Path(directory) / "app.env"
            env.write_text("APP_ENV=production\nADMIN_PASSWORD='" + marker + "'\nMONGODB_URI=mongodb+srv://user:" + marker + "@fixture.invalid\n", encoding="utf-8")
            output = io.StringIO()
            with patch("sys.argv", ["security-check", "--env-file", str(env)]), patch.object(check, "capture", return_value=(False, marker)), patch("sys.stdout", output):
                check.main()
            self.assertNotIn(marker, output.getvalue())
            report = json.loads(output.getvalue())
            self.assertTrue(report["read_only"])
            self.assertTrue(report["configuration"]["admin_password_not_template"])
            self.assertEqual(check.read_environment(env)["ADMIN_PASSWORD"], marker)

    def test_unsafe_mongo_and_template_flags(self):
        """Flag explicit TLS bypasses and documented template credentials."""
        for option in ("tls=false", "ssl=false", "tlsInsecure=true", "tlsAllowInvalidCertificates=true", "tlsAllowInvalidHostnames=true"):
            flags = check.configuration_checks({"MONGODB_URI": "mongodb+srv://fixture.invalid/?" + option, "ADMIN_PASSWORD": "REPLACE_WITH_PRIVATE_PASSWORD_AT_LEAST_8_CHARACTERS"})
            self.assertFalse(flags["mongo_tls_enabled_without_insecure_overrides"])
            self.assertFalse(flags["admin_password_not_template"])
            self.assertFalse(flags["payment_webhook_secret_configured"])
        self.assertFalse(check.configuration_checks({"RAZORPAY_WEBHOOK_SECRET": "REPLACE_ME"})["payment_webhook_secret_configured"])
        self.assertTrue(check.configuration_checks({"RAZORPAY_WEBHOOK_SECRET": "isolated-private-fixture"})["payment_webhook_secret_configured"])

    def test_proxy_transformation_preserves_tls_and_is_idempotent(self):
        """Keep existing HTTPS directives while overwriting visitor-supplied addresses in every known proxy location."""
        original = "server {\n listen 443 ssl;\n ssl_certificate /private/certificate;\n location / {\n  proxy_pass http://127.0.0.1:8080;\n  proxy_set_header X-Real-IP $http_x_real_ip;\n }\n location /api/auth/ {\n  proxy_pass http://127.0.0.1:8080;\n }\n}\n"
        hardened = proxy.proxy_configuration(original)
        self.assertEqual(hardened.count("X-Real-IP $remote_addr;"), 2)
        self.assertNotIn("$http_x_real_ip", hardened)
        self.assertIn(" ssl_certificate /private/certificate;", hardened)
        self.assertEqual(proxy.proxy_configuration(hardened), hardened)
        for custom in ("proxy_pass http://remote-server;", "location / { proxy_pass http://127.0.0.1:8080; }"):
            with self.assertRaises(ValueError):
                proxy.proxy_configuration(custom)

    def test_environment_flag_preserves_other_values(self):
        """Replace only the trust flag and preserve quoted password syntax exactly."""
        original = "ADMIN_PASSWORD='private fixture # value'\nTRUST_LOOPBACK_PROXY=false\n"
        changed = proxy.environment_configuration(original)
        self.assertEqual(changed, "ADMIN_PASSWORD='private fixture # value'\nTRUST_LOOPBACK_PROXY=true\n")
        self.assertEqual(proxy.environment_configuration(changed), changed)

    def test_failed_reload_restores_both_files(self):
        """Simulate a failed Nginx reload and verify rollback restores the original environment before returning failure."""
        metadata = SimpleNamespace(st_uid=0, st_gid=0, st_mode=0o100600)
        nginx = unittest.mock.MagicMock()
        env = unittest.mock.MagicMock()
        backup = unittest.mock.MagicMock()
        nginx.stat.return_value = env.stat.return_value = metadata
        nginx.is_symlink.return_value = env.is_symlink.return_value = False
        nginx.read_text.return_value = "proxy_pass http://127.0.0.1:8080;\n"
        env.read_text.return_value = "TRUST_LOOPBACK_PROXY=false\n"
        writes = []
        # Fail after initial validation and new configuration validation, at reload.
        with patch.object(proxy.os, "geteuid", return_value=0, create=True), patch.object(proxy, "Path", side_effect=[nginx, env, backup]), patch.object(proxy.tempfile, "mkdtemp", return_value="fixture"), patch.object(proxy.shutil, "copy2"), patch.object(proxy, "replace_file", side_effect=lambda path, content, meta: writes.append((path, content))), patch.object(proxy, "checked_command", side_effect=[None, None, RuntimeError("fixture"), None, None]):
            with self.assertRaises(RuntimeError):
                proxy.main()
        self.assertEqual(writes[-2:], [(nginx, nginx.read_text.return_value), (env, env.read_text.return_value)])


if __name__ == "__main__":
    unittest.main()
