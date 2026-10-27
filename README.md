# Mishra 4 You · पं. राजा मिश्रा

An English-default astrology application with an English/Hindi language selector: React + TypeScript + Vite, a Go modular monolith, the existing Swiss Ephemeris CGO engine, and **MongoDB Atlas** (changed from PostgreSQL at the user's request). No public deployment or real payment/message has been performed.

For code navigation, start with the [feature and folder map](docs/CODEBASE.md),
the searchable [function index](docs/FUNCTION_INDEX.md), and the
[contribution guide](CONTRIBUTING.md). Maintained named functions have purpose
comments; `npm --prefix web run docs:check` checks coverage and index freshness.

For a simple-English explanation of the implemented calculations, read
[How Our Software Calculates Results](docs/CALCULATION-METHODS-SIMPLE.md).

The 19 October 2026 critical calculation audit covers the implemented natal,
Panchang, sky, scheduling and financial calculations. Independent NASA/JPL
phase fixtures and NASA eclipse catalogs supplement the official Swiss CLI
references. Current calculation versions are Kundli v3, Panchang v2, sky v2
and Panchang rules v2. See the [calculation contract and accuracy limits](docs/CALCULATIONS.md)
and [verification record](docs/VERIFICATION.md); passing samples do not imply
complete provider or regional ritual parity.

The language selector is available in the header and dialogs. English is the default regardless of browser language; a saved Hindi preference is respected on later visits. Switching preserves birth details, reports and dialog inputs. UI text, default FAQ/policies, chart labels, accessibility labels and account dates follow the selected language. Approved interpretations use their supplied English version when available; original approved content and existing paid PDF snapshots are preserved.

The public website, customer login and administration use separate pages and
layouts. The public header links to customer login and does not expose admin navigation.

| Route | Purpose |
| --- | --- |
| `/` | Public landing page and daily tools |
| `/login` | Standalone customer mobile/OTP login |
| `/account` | Protected customer reports, bookings and privacy settings |
| `/kundli` | Current guest Kundli, retained only in this browser tab |
| `/kundli-matching` | Public two-person Ashtakoota comparison; no login or saving |
| `/vivah-muhurat` | Public monthly marriage shortlist with India city, timing evidence and exclusion reasons |
| `/bhakti` | Public devotional library with text readers, related PNG images, 12-card pagination and six categories, including Kavacham |
| `/bhakti/:category/:slug` | Shareable prayer reader with adjustable text size |
| `/shop` | Gemstone catalogue with search and category filters |
| `/shop/stones/:slug` | Stone details, photo and certificate information |
| `/shop/bag` | Cart, Indian delivery address, order review and payment |
| `/shop/orders` | Customer gemstone purchases and delivery tracking |
| `/account/reports/:id` | Protected, reloadable saved report |
| `/admin/login` | Separate administrator password/TOTP login |
| `/admin` | Protected administration console |

The main-page Gemstones menu opens `/#gemstones`, an embedded collection of up
to six stones from the public catalogue. Product links lead to their details;
the full shop, cart and checkout remain available through the existing routes.

The shop now includes 16 editable reference-priced listings: the nine Navratna
stones and amethyst, white sapphire, opal, citrine, turquoise, peridot and
moonstone. Individual pages include image zoom, quantity selection, cart actions,
bilingual stone guides, traditional associations, care and disclosures.
Initial prices were researched on 20 October 2026; the current displayed price
is controlled through the admin editor. Customer pages show no AI image badges,
indicative-price captions or historical retail-price panels. The homepage and
shop cards also support Add to cart.

To edit them, open Administration → Gemstone shop, select a stone and change its
price per piece, weight, description, guide, photo or quantity, then Save stone.
Starter entries require stock confirmation: the cart works, while payment
requires confirmed inventory. Clear the stock-confirmation checkbox only after
updating the actual stock disclosures. Merchant shipping, tax and checkout
settings remain separately configurable. `go run ./cmd/shopcatalog` adds missing
starter slugs in development without replacing manual edits or merchant settings.

Customer and administrator cookies remain independent. Browser Back/Forward and
refresh work with these URLs. Saving a Kundli or booking a consultation resumes
after login; return destinations are restricted to local customer routes. See
[route and layout notes](docs/ROUTES.md).

New v4 Kundli reports include actual-birth Panchang with full start/end
intervals, a birth-star profile, Sripati Bhav Chalit, all sixteen Parashari
vargas from D1 through D60 and a Moon chart. Planet analysis includes sign
dignity, combustion, Baladi states, classical aspects, three friendship tables,
auditable Vimshopaka weights, BAV/SAV and Prastara Ashtakavarga tables.
Five-level Vimshottari timelines (Mahadasha, Antardasha, Pratyantar, Sukshma
and Prana) and Yogini/Antardashas are available. The single-person Kundli
reuses the audited Lagna/Moon/Venus Mangal assessment and adds eleven yoga
placement patterns. New calculations also include a versioned Kaal Sarp
enclosure check with twelve house-based types and basic Kendra–Trikona Rajyog checks.
The Numerology section records Mulank (birth day), Bhagyank (full birth date)
and Chaldean Naamank (Latin name spelling), their reductions and number rulers.
Naamank uses the main birth name, with no additional spelling field. Unsupported
name scripts have an availability note. All nine Mulank profiles include original Hindi/English nature,
work, surroundings, timing, routine, devotional and remedy readings, favourable
correspondences and a planetary number yantra. Facts and prose are snapshotted;
gemstone associations are not personal wearing prescriptions.
It also works when birth time is unknown. The arithmetic and method are stored
with new reports and printed in remedy PDFs; old snapshots are not backfilled.
English/Hindi panels and selected-language PDF downloads lead with a personal
conclusion and a plain-language traditional meaning, followed by the supporting
positions and lord relationships. Absent, undecided and cancelled indications
remain distinct; older reports need recreation for missing calculations.
Purchased remedy PDFs offer an English/Hindi selector in the account. English
downloads use the purchased English interpretation where supplied; original
mantras and remedy instructions retain their saved wording. Existing Hindi
files and purchase snapshots stay intact.
The PDF opens as an illustrated book with a birth portrait, topic-by-topic
personal readings, plain-language Yoga/Dosha conclusions, numerology and a Dasha
reference. Full calculation tables follow in numbered appendices. Original
Hindi/English chapter writing, linked contents, bookmarks and a restrained
cream/burgundy design make the reading distinct from the technical evidence.
Transits retain their timestamped snapshot, with an
on-demand Sade Sati/Dhaiya timeline including retrograde exits/re-entries and
labelled clipped edges. Faladesh reads
applicable Guru Ji-approved content; missing approvals are shown transparently.
Admin rules support life-topic categories and compound chart conditions. See
[calculation methods and supported limits](docs/CALCULATIONS.md). New remedy
PDFs now print the available expanded snapshot, including birth Panchang,
Moon/Chalit/varga charts, planet analysis, Ashtakavarga, yoga/Mangal evidence,
all retained Vimshottari/Yogini parents and Antardashas, the running five-level
sequence and timestamped transits. Guidance and supplied remedies appear first,
with repeating decorative borders, page numbers and linked contents. Older
purchases keep their original snapshots; interactive date-window Saturn timelines
and arbitrary deeper Dasha branches remain on the website. See
[PDF scope and verification](docs/REMEDIES-PDF.md). The Kundli Dasha explorer shows both
clearly labelled start and end dates/times to the minute in the birthplace's timezone, a
running five-level sequence and inline, expandable planet rows. Open a planet
to reveal nine subperiods beneath it, retaining the full sequence (such as
Saturn–Saturn–Saturn) and every parent's start/end time through Prana.
Birth forms display hours/minutes. Stored/API instants retain their precision;
date/time accuracy depends on the recorded inputs and documented year convention.

Kundli matching is available at **/kundli-matching** in English/Hindi. Enter
both known birth times and select trusted birthplaces, with birth dates from
1940 through today at each birthplace. `POST /api/kundli/match` calculates the
eight base Ashtakoota factors out of 36 and separate three-reference Mangal analysis.
It requires no account, saves neither partner nor result, and calls no paid
provider. The page holds the comparison only during the current visit and
has Guna Milan, Mangal, D1/D9/Moon chart and birth-detail tabs, with factual
factor explanations and English/Hindi PDF download. PDF export recalculates
the inputs server-side. Signed-in customers can explicitly select one of
their existing saved Kundlis; guest matching never requires a login or saves
either person's details. The versioned AIFAS table convention, its explicit Cow/Tiger
cell correction and exclusions are documented in [CALCULATIONS.md](docs/CALCULATIONS.md#kundli-matching).

The existing video section contains the ten most-viewed standard videos from
[Mishra 4 You](https://www.youtube.com/channel/UC9-N4M0ZcIVefwMx4_Ep7TA),
selected from YouTube's Popular tab and checked on 18 October 2026. Titles,
views, durations and publication dates are bundled in
`internal/platform/data/channel-videos.json`; counts are a dated snapshot.
English uses concise editorial translations of titles; Hindi retains the
original YouTube titles. Players load on click, with direct YouTube links.
The admin editor supports up to ten unique IDs in the updated backend.
`go run ./cmd/channelvideos` applies this selection to development storage
without changing other content, settings or approval flags.

Birthplace search includes about 549,000 Indian cities, towns and villages across all 36 states/union territories, with alternate names and district/state qualifiers. The bundled [GeoNames catalog](internal/platform/data/README.md) needs no API key; coverage varies and does not guarantee every village. The browser fetches at most 25 matches at a time. Existing international choices and saved birthplace IDs remain available.

The refreshed visual design uses a burgundy opening with custom SVG celestial orbits, saffron accents, warm-white surfaces and upright DM Sans with Noto Devanagari. Responsive tools navigation, a scrollable ten-video library and the Kundli reading workspace share this identity. Fonts are hosted locally with their license notices. [Design notes](docs/DESIGN.md) describe the reference-site review, responsive layouts and stylesheet organization. The interface uses no italic styling.

The public **Puja Vidhi** menu opens `/puja-vidhi` with 27 brief English/Hindi
Puja, Vrat and Homa guides, twelve-card paging, category/name search, local PNG
artwork, materials, numbered worship steps and related Aarti links. Guide bodies
load from the site's own JSON asset only when opened; references and image
licences remain available. [Guide notes](docs/PUJA-VIDHI.md) describe the original
editorial scope and maintenance commands. No video or sign-in is required.

The landing page's native Panchang provides location/date selection, all five
Panchang elements (weekday plus Tithi, Nakshatra, Yoga and Karana), precise end
times, sunrise/sunset, Rahu Kaal, Yamaganda, Gulika, Abhijit, Choghadiya and
Hora in English/Hindi. It uses our Swiss engine and the India place catalog,
with no external Panchang API key. Its Bhagwa overview opens with four essential
values and solar times; all intervals and extended details are available in two
optional disclosures that mount on first expansion and retain later selections.
The read-only endpoint is
`GET /api/panchang?date=2026-10-18&place_id=delhi&sunrise=upper-limb`;
`sunrise=hindu` selects the geometric Hindu convention. The same panel now
includes **Planets** (positions, transits, combustion and retrograde) and **Eclipses** (previous
and next solar/lunar events relative to now). Positions include nine traditional Grahas,
Lahiri sign/degrees, Nakshatra/Pada, speed and retrograde status at a chosen
local time. Transits include direct/retrograde sign entries and motion
stations, with year/planet filters. Retrograde has a dedicated station-only view
for Mars through Saturn. Combustion lists annual start/end boundaries, Sun-longitude
separation and the applied limit, using the same declared rules as Kundli analysis;
Sun and nodes are excluded. Both views have bilingual controls, empty/error/retry
states and twenty-result paging. The compact eclipse view shows each kind's
last/next date, global type and maximum time, with selected-city visibility
and any differing local type. Adjacent years are searched automatically;
annual lists, visibility filters and detailed contacts are not displayed.
Partial, total, annular, hybrid and penumbral types are supported. Calculations are
stateless, rate-limited and cached in bounded per-process caches. Sky endpoints
are `/api/sky/positions?date=2026-10-18&time=12:00&place_id=delhi`,
`/api/sky/transits?year=2026&place_id=delhi`,
`/api/sky/combustion?year=2026&place_id=delhi` and
`/api/sky/eclipses?year=2026&place_id=delhi`. Birth dates start on 1 January 1940;
daily tools support 1 January 1940–30 December 2099 and annual tools support 1940–2099;
all displayed dates/times use IST. Festival
calendars and event-specific Muhurat are outside this daily-tools increment.

## Run locally

Requirements: Go 1.26.5+, a C compiler for CGO, Node 22.12+ (tested with 22.17), and Chrome/Chromium for PDF rendering. Windows GCC is supported. The bundled ephemeris files and Noto Devanagari fonts are required.

From the repository root:

```powershell
Copy-Item .env.example .env
# Edit .env privately. Put your Atlas URI in MONGODB_URI.
go mod download
go run ./cmd/server
```

In a second terminal:

```powershell
cd web
npm ci
npm run dev
```

Open **http://localhost:5173**. The API listens at http://127.0.0.1:8080. Both localhost and 127.0.0.1 frontend origins work in development. Production accepts only the exact configured HTTPS origin.

The Panchang includes native extended calculations in a daily-only view.
Weekly and monthly browsing have been removed. See `docs/CALCULATIONS.md` for the supported astronomy, traditional rules,
reference checks and explicit exclusions. The Atlas-backed API was restarted on
19 October 2026 with the updated Panchang binary; daily Panchang is available at
**http://localhost:5173/#panchang**. An existing Go process must be restarted after
backend source changes; Vite alone only reloads the frontend. The isolated
file-store preview remains separate at **http://127.0.0.1:18082/#panchang**.

The root `.env` is automatically loaded without overriding existing environment variables. `ENV_FILE=none` disables file loading for isolated tests; another path selects a different environment file. Do not put secrets in `web/.env`, frontend code or variables beginning `VITE_`. `.env` is ignored. No application log prints the MongoDB URI.

```dotenv
MONGODB_URI=your-private-atlas-uri
MONGODB_DATABASE=mishra4you_development
```

Use a separate development database and a database user limited to that database. Add your current development machine's IP to Atlas Network Access. URL-encode special characters in the database password. The database name comes from `MONGODB_DATABASE`, not an implicit database in the URI. Atlas supplies the replica-set transactions required by booking/payment invariants. Standalone MongoDB is rejected.

With `MONGODB_URI` blank, the app uses `.local/development-state.json`. This is a labelled, single-process development fallback; production refuses it. Existing local demo data is not automatically imported into Atlas. Connecting Atlas creates the app's empty development database/indexes and demo draft content, without copying private local test reports.

## Try the working flows

- Create a guest Kundli with a listed city and a birth date from 1940 through today. See the real calculation, chart, table, settings and limitations before login.
- Open `/kundli-matching`, enter both birth details and review all eight base scores and the separate Mangal geometry. No login or saved report is created.
- Save it using mobile OTP. In local simulation, the dialog displays a randomly generated development code; no SMS is sent. The form survives authentication and tab refresh.
- Choose the ₹99 PDF. The server snapshots the calculation/rules and creates a 9,900-paise order. Explicitly confirm the **local payment simulation**. The private Hindi PDF becomes available in the account; the worker retries failures without another purchase.
- Book the ₹3,501 consultation independently. The order is 350,100 paise. Choose an available time, obtain a ten-minute hold and confirm the simulated payment. Simulated Zoom does not invent an attendee link. Reschedule/cancel in the account.
- Open **http://localhost:5173/admin**. Local-only default admin password: `local-admin-change-me`. Set `ADMIN_PASSWORD` to change it. Production requires a password of at least 20 characters and an authenticator TOTP secret; the default password is refused.
- Admin opens a business overview with monthly gross collections, paid stone orders awaiting dispatch, upcoming consultations, stock attention and delivery retries. Grouped sidebar navigation (a section selector on phones/tablets) reaches products/prices, payments, consultations, customers, finances, content, interpretations, jobs, settings and audit history. Product/customer/order search and payment-status filters help find records. Operational order/booking summaries exclude tests; finance uses server totals for the selected month.
- Admin manages settings, availability, blocked dates, versioned content, jobs, orders/refunds, expenses with private receipt uploads, monthly reconciliation, recorded distributions and CSV exports. Customer phones are masked in lists. There are only two roles in this release: customer and owner/admin; there are no delegated staff accounts.

The session duration is intentionally **unset**. Local scheduling uses a labelled 30-minute example; live bookings require a configured duration. Approved portrait, introduction, videos, remedies and policies are not invented. Demo content cannot be approved as paid production content.

## Build and check

```powershell
go test ./...
go vet ./...
cd web
npm run build
npm run test:e2e
```

Browser tests build the frontend and automatically launch their own Go server on localhost port 18081, with `ENV_FILE=none`, a unique local data directory and all providers in simulation. They never load the application's Atlas URI. Set `CHROME_PATH` if Chrome is not installed in its default Windows location. `E2E_URL` can select another unused localhost port. They cover the actual UI journey, phone/tablet/desktop layouts, overflow and Axe accessibility checks.

Real PDF rendering test (PowerShell, root):

```powershell
$env:RUN_PDF_TEST='1'
go test ./internal/platform -run TestHindiPDFRenderer -v
```

MongoDB integration tests are opt-in and use only `MONGODB_TEST_URI`, never `MONGODB_URI` from `.env`. They create a randomly named `m4y_test_*` database and remove that database after the test:

```powershell
# Set MONGODB_TEST_URI privately to a dedicated replica set.
go test ./internal/platform -run TestMongoTransactionsAcrossConnections -v
```

For a workspace-local test replica set on port 27028, `MONGODB_TEST_INIT=true` permits test initialization of the localhost-only `m4ytest` replica set. That initialization is never allowed on Atlas or another hostname.

## Source layout

- `cmd/server`: HTTP service and migration command; `cmd/kundalicheck`: preserved calculation CLI.
- `internal/swisseph`: preserved C engine, mutex-protected Go integration and reference fixtures.
- `internal/platform`: accounts, reports, payments, scheduling, content, accounting, providers, private PDF worker, MongoDB store and tests.
- `internal/platform/migrations`: versioned MongoDB index migrations (no PostgreSQL dependency).
- `web`: responsive Hindi customer and admin application.
- `assets`: bundled Noto font files and SIL licence. Static regular/bold instances are included for embedded-font PDFs; Python is not a backend/runtime dependency.
- `deploy`, `docs`: deployment, operations, integration contracts and verification evidence.

## Before live use

See [integration configuration](docs/INTEGRATIONS.md), [deployment and operations](docs/OPERATIONS.md), [calculation contract](docs/CALCULATIONS.md), [scope](docs/PLAN.md) and [verification results](docs/VERIFICATION.md).

Awaiting external configuration: Atlas URI/network access, Razorpay test credentials/webhook, SMS gateway and approved OTP template, Zoom Server-to-Server OAuth account, notification gateway. Awaiting business approvals: Swiss Ephemeris licensing choice, applicable tax treatment, session duration/availability, policies/support contact, Guru Ji portrait/introduction/videos and approved applicable interpretation/remedy coverage. Live service stays blocked until the relevant settings and content are ready.

Infrastructure cost of ₹1,700/month is a planning estimate, not an automatically booked expense or a promise covering providers, licences, Atlas tiers or future traffic.
