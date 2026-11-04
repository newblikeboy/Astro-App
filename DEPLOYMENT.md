# Build, run and deploy Mishra4You

This guide is part of the corresponding source offered under [LICENSE](LICENSE).
It describes the application's build/install process and external service
interfaces. All commands below use public source and placeholder credentials.

## Requirements and source versions

- Go **1.26.5** with `GOTOOLCHAIN=go1.26.5` and `GOEXPERIMENT=nojsonv2`.
  The pinned go-json-experiment dependency fails against Go 1.27.3's JSON v2
  API. Do not substitute the latest Go release without testing dependencies.
- A C compiler and CGO enabled; Swiss Ephemeris C sources are bundled in
  internal/swisseph. Linux builds link libm and libdl.
- Node.js 22.12+ and npm; Node 22.17 is the development reference.
- Chrome/Chromium for local PDF rendering, run as a non-root user with its
  browser sandbox available.
- Bundled ephe/*.se1 files, assets/NotoSansDevanagari*.ttf and web/public assets.
- MongoDB Atlas or another supported replica set for production transactions.

Clone <https://github.com/newblikeboy/Astro-App>. A clean checkout of a published
commit is the build input. The frontend embeds that commit's SHA in its visible
source, licence and notices links. Publish changes before deploying them; keep
old deployed commits available. Never build a production deployment from
uncommitted changes or delete the published source of a deployed release.

For a source archive without .git, obtain the full 40-character SHA associated
with that archive and set `SOURCE_REVISION=<sha>` before `npm run build`.
This value is public build metadata, not a credential. Do not point it at a
different version. Changing frontend or backend source requires publishing a
new revision and rebuilding both from that revision.

## Local development

From a cloned checkout, on Linux/macOS:

```bash
export GOTOOLCHAIN=go1.26.5
export GOEXPERIMENT=nojsonv2
cp .env.example .env
go mod download
CGO_ENABLED=1 go run ./cmd/server
```

In a second terminal:

```bash
cd web
npm ci --include=dev
npm run dev
```

Open http://localhost:5173. The API listens on 127.0.0.1:8080. With no
MONGODB_URI the development-only file store uses DATA_DIR; production refuses
this storage mode. The .env.example defaults to simulated integrations that
make no real payment, SMS or Zoom call. An OTP shown in development is not a
production SMS service. Never expose this development configuration publicly.

On PowerShell set `$env:GOTOOLCHAIN='go1.26.5'` and
`$env:GOEXPERIMENT='nojsonv2'`, and use `Copy-Item .env.example .env`.

## Fresh Ubuntu 24.04 x64 server

The following setup commands assume a fresh server and root SSH access.
The app itself runs under a dedicated non-root system account. A 1 vCPU/1 GB
server can be memory constrained during builds and PDFs; swap is a fallback,
not a capacity guarantee. Test concurrent traffic and PDF rendering separately.

```bash
apt-get update
apt-get install -y git build-essential curl ca-certificates python3 openssl \
  nginx certbot python3-certbot-nginx ufw

if [ ! -e /swapfile ]; then
  fallocate -l 2G /swapfile
  chmod 600 /swapfile
  mkswap /swapfile
  swapon /swapfile
  echo '/swapfile none swap sw 0 0' >> /etc/fstab
fi

# Install into /usr/local on a fresh server without an existing Go installation.
curl -fL https://go.dev/dl/go1.26.5.linux-amd64.tar.gz -o /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
export PATH="/usr/local/go/bin:$PATH"
echo 'export PATH=/usr/local/go/bin:$PATH' > /etc/profile.d/golang.sh
export GOTOOLCHAIN=go1.26.5
export GOEXPERIMENT=nojsonv2

curl -fsSL https://deb.nodesource.com/setup_22.x -o /tmp/nodesource.sh
bash /tmp/nodesource.sh
apt-get install -y nodejs

curl -fL https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
  -o /tmp/google-chrome.deb
apt-get install -y /tmp/google-chrome.deb

git clone --branch main https://github.com/newblikeboy/Astro-App.git /opt/mishra4you
useradd --system --user-group --home-dir /var/lib/mishra4you \
  --shell /usr/sbin/nologin mishra4you
install -d -m 700 /etc/mishra4you
install -d -o mishra4you -g mishra4you -m 700 /var/lib/mishra4you

cd /opt/mishra4you
go mod download
mkdir -p bin
CGO_ENABLED=1 GOMAXPROCS=1 GOMEMLIMIT=450MiB \
  go build -p 1 -trimpath -o bin/mishra4you ./cmd/server
cd web
npm ci --include=dev
NODE_OPTIONS=--max-old-space-size=512 npm run build
cd ..
```

### Configure storage and providers

Copy the production example once; do not overwrite an existing configuration:

```bash
test ! -e /etc/mishra4you/app.env && \
  install -m 600 deploy/app.env.example /etc/mishra4you/app.env
nano /etc/mishra4you/app.env
```

Replace every placeholder privately. APP_ORIGIN is the exact HTTPS origin,
without a trailing slash. Set MONGODB_DATABASE explicitly and permit the
droplet's IP in Atlas. The database user needs read/write and index-creation
permissions for that database; startup applies embedded migrations.
The database must support replica-set or sharded transactions.

Set ADMIN_EMAIL to the administrator's email address and ADMIN_PASSWORD to the
chosen private password (at least 8 characters). `openssl rand -hex 24` can
generate a random password. Keep both settings in /etc/mishra4you/app.env;
never place the password in frontend code or commit it to Git. Sign-in uses
email and password; no authenticator code is required. Legacy ADMIN_TOTP_SECRET
settings can be removed. Production also requires MongoDB, an HTTPS origin
and the provider settings below.

To set or change those credentials without putting the password in shell
history or Git, use the included helper. It prompts for the password twice,
retains other environment settings and writes the file with mode 0600:

```bash
python3 deploy/set-admin-login.py --email admin@your-domain.example
systemctl restart mishra4you
```

For an existing deployment migrating from password/TOTP login, pull the updated
source and build the updated frontend and backend first. Then run the helper,
install the new backend binary and restart the service. Supply your own
administrator email in place of the example.

- **Razorpay:** PAYMENT_MODE=razorpay and all three RAZORPAY_* values. Start
  with test keys; ALLOW_LIVE_PAYMENTS=false keeps live keys disabled. Webhook:
  POST /api/payments/webhook. Preserve its body for signature verification.
  Configure captured/failed payment and refund events in the provider account.
- **OTP:** OTP_MODE=http, HTTPS OTP_GATEWAY_URL and OTP_GATEWAY_TOKEN.
  The gateway receives an application/json POST with bearer authentication:
  `{"phone":"+91...","code":"123456","purpose":"authentication","expires_in":"300"}`.
  A 2xx response means the gateway accepted the SMS request. A provider-specific
  adapter may be needed; a vendor's API URL is not automatically compatible.
- **Zoom:** ZOOM_MODE=zoom, ZOOM_ACCOUNT_ID, ZOOM_CLIENT_ID, ZOOM_CLIENT_SECRET
  and ZOOM_USER_ID for the configured host. Use Server-to-Server OAuth with
  meeting create/read/delete permissions appropriate to that host. Provider
  account capabilities and quotas remain external configuration.
- **Notifications:** HTTPS NOTIFICATION_URL and NOTIFICATION_TOKEN. An
  application/json POST with bearer authentication sends
  `{"phone":"+91...","template":"pdf_ready","idempotency_key":"unique-job-key"}`.
  Supported templates are pdf_ready, booking_confirmed, booking_reminder and
  booking_cancelled. The gateway must deduplicate idempotency keys and return
  2xx on acceptance.
- **Private files:** DATA_DIR is writable by mishra4you. Fonts and ephemeris
  stay readable in /opt/mishra4you. CHROME_PATH selects the installed browser.
  Keep the browser sandbox enabled; do not run the application as root.

### Start the service

```bash
cp deploy/mishra4you.service /etc/systemd/system/mishra4you.service
systemctl daemon-reload
systemctl enable --now mishra4you
curl --fail --retry 15 --retry-connrefused --retry-delay 2 \
  http://127.0.0.1:8080/api/health
test -s web/dist/index.html
curl --fail -I http://127.0.0.1:8080/
```

The Go service serves both /api and the built frontend. Missing web/dist/index.html
causes a homepage 404 even when /api/health succeeds. Inspect startup failures
with `journalctl -u mishra4you -n 80 --no-pager`; never publish app.env.

### Nginx, domain and HTTPS

Point both the apex and www DNS records to this server; remove stale IPv6
records. Substitute your own domain when deploying a fork. This fresh-server
example uses mishra4you.com and the repository's Nginx template:

```bash
sed 's/replace-with-approved-domain/mishra4you.com www.mishra4you.com/g' \
  deploy/nginx.conf.example > /etc/nginx/sites-available/mishra4you
ln -s /etc/nginx/sites-available/mishra4you /etc/nginx/sites-enabled/mishra4you
nginx -t && systemctl reload nginx

# Assumes SSH is on port 22. Allow your actual SSH port before enabling UFW.
ufw allow OpenSSH
ufw allow 'Nginx Full'
ufw --force enable

certbot --nginx -d mishra4you.com -d www.mishra4you.com --redirect
```

After certificate issuance succeeds, add an apex-domain redirect for www:

```bash
sed -i '/^[[:space:]]*server_name .*mishra4you\.com.*;/a\    if ($host = www.mishra4you.com) { return 301 https://mishra4you.com$request_uri; }' \
  /etc/nginx/sites-available/mishra4you
nginx -t && systemctl reload nginx
curl --fail https://mishra4you.com/api/health
curl --fail -I https://www.mishra4you.com/
certbot renew --dry-run
```

An attached cloud firewall must also permit SSH, HTTP and HTTPS. Keep
APP_ORIGIN=https://mishra4you.com; redirecting www keeps browser origins and
authentication consistent. Do not overwrite a live TLS configuration with
the fresh-server template on subsequent updates.

### Operator confirmation

After publishing/deploying the AGPL source and notices and fulfilling the
applicable terms, the operator can confirm the Swiss Ephemeris licensing
choice in Administration > Service settings. This is an application setting,
not registration with Astrodienst. It is deliberately not auto-enabled by this
source update. Other approvals/settings may independently block paid features.

Current guest-Kundli code reports "storage unavailable" for both a failed
database read and an unset licence-confirmation setting; a successful health
check alone does not distinguish these cases. Review the admin setting first.

## Updating an existing deployment

Push the complete source before pulling it on the server. From a clean
checkout (keep the build settings in every deployment session):

```bash
(
  set -e
  export PATH="/usr/local/go/bin:$PATH"
  export GOTOOLCHAIN=go1.26.5
  export GOEXPERIMENT=nojsonv2
  cd /opt/mishra4you
  test -z "$(git status --porcelain)"
  git pull --ff-only origin main
  git rev-parse HEAD
  CGO_ENABLED=1 GOMAXPROCS=1 GOMEMLIMIT=450MiB \
    go build -p 1 -trimpath -o bin/mishra4you.new ./cmd/server
  cd web
  npm ci --include=dev
  NODE_OPTIONS=--max-old-space-size=512 npm run build
  cd ..
  mv bin/mishra4you.new bin/mishra4you
  systemctl restart mishra4you
  curl --fail --retry 15 --retry-connrefused --retry-delay 2 \
    http://127.0.0.1:8080/api/health
)
```

The source links identify this build's published commit, even after main
advances. Preserve commits/release archives. For a fork, update the repository
URL in web/src/SourceLinks.tsx to the public corresponding-source repository.
The in-place update has a brief window where frontend/backend versions differ;
use a maintenance window or versioned release directories when this matters.

## Verification, backups and maintenance

For code changes use `ENV_FILE=none go test ./...`, `go vet ./...` and
`npm --prefix web run build` with the documented toolchain. Existing Playwright
tests use an isolated local development server; they must not target production.

Verify homepage, account/admin login, sandbox payment/webhook handling, PDF
generation/download and booking/provider behaviour before enabling real sales.
Back up Atlas and private DATA_DIR files off-host, retain secrets separately,
and test restores to an isolated database. Monitor memory/disk, service logs,
certificate renewal and queued jobs; patch dependencies/browser/OS after
testing the supported build. Credentials, customer records and private PDFs
are operational data and must not be included in the public source offer.
