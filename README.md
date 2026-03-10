# Nurix

A command-line tool to manage [Caddy](https://caddyserver.com/) reverse proxy records backed by PostgreSQL.

Instead of manually editing your Caddyfile, you register domains and DNS records through **nurix**. The Caddyfile is auto-generated from the database after every change and Caddy is reloaded automatically.

---

## Table of Contents

- [Nurix](#nurix)
  - [Table of Contents](#table-of-contents)
  - [Why Nurix?](#why-nurix)
  - [Features](#features)
  - [Architecture](#architecture)
  - [Prerequisites](#prerequisites)
    - [Create the database](#create-the-database)
  - [Installation](#installation)
    - [Build from Source](#build-from-source)
    - [Using the Makefile](#using-the-makefile)
  - [Getting Started](#getting-started)
    - [Step 1 — Configure](#step-1--configure)
    - [Step 2 — Run Migrations](#step-2--run-migrations)
    - [Step 3 — Register Domains](#step-3--register-domains)
    - [Step 4 — Add DNS Records](#step-4--add-dns-records)
  - [Commands Reference](#commands-reference)
    - [nurix set config](#nurix-set-config)
    - [nurix run db-migration](#nurix-run-db-migration)
    - [nurix domain add](#nurix-domain-add)
    - [nurix domain update](#nurix-domain-update)
    - [nurix domain remove](#nurix-domain-remove)
    - [nurix domain search all](#nurix-domain-search-all)
    - [nurix dns add](#nurix-dns-add)
    - [nurix dns update](#nurix-dns-update)
    - [nurix dns remove](#nurix-dns-remove)
    - [nurix dns search all](#nurix-dns-search-all)
    - [nurix version](#nurix-version)
    - [nurix help](#nurix-help)
  - [Generated Caddyfile](#generated-caddyfile)
  - [Database Schema](#database-schema)
    - [domains](#domains)
    - [dns_records](#dns_records)
    - [changelog](#changelog)
    - [schema_migrations](#schema_migrations)
  - [Validation Rules](#validation-rules)
  - [Audit Trail](#audit-trail)
  - [Security](#security)
    - [Credential Storage](#credential-storage)
    - [Database](#database)
    - [Caddyfile](#caddyfile)
  - [Project Structure](#project-structure)
  - [Development](#development)
    - [Clone and build](#clone-and-build)
    - [Run locally without installing](#run-locally-without-installing)
    - [Adding a new migration](#adding-a-new-migration)
    - [Run tests](#run-tests)
  - [Troubleshooting](#troubleshooting)
    - ["nurix is not configured yet"](#nurix-is-not-configured-yet)
    - ["failed to connect to database"](#failed-to-connect-to-database)
    - ["no registered domain found for 'app.example.com'"](#no-registered-domain-found-for-appexamplecom)
    - ["domain 'example.com' expired on 2025-01-01"](#domain-examplecom-expired-on-2025-01-01)
    - ["cannot remove domain — it still has X DNS record(s)"](#cannot-remove-domain--it-still-has-x-dns-records)
    - ["Caddyfile written but reload failed"](#caddyfile-written-but-reload-failed)
    - [Keyring issues on headless Linux servers](#keyring-issues-on-headless-linux-servers)
  - [License](#license)

---

## Why Nurix?

Managing a Caddy reverse proxy on a VPS usually means SSH-ing in and hand-editing the Caddyfile every time you deploy a new service. This approach:

- Is error-prone (typos break all sites)
- Has no audit trail (who changed what and when?)
- Has no validation (pointing to an expired domain?)
- Has no structure (Caddyfile becomes a mess with 20+ entries)

**Nurix** solves all of this by making PostgreSQL the single source of truth and treating the Caddyfile as a generated artifact.

---

## Features

- 🗄️ **PostgreSQL as source of truth** — all records live in the database, the Caddyfile is generated
- 🔐 **Secure credential storage** — config (DB password, etc.) stored in OS keyring, not on disk
- 📋 **Domain management** — register domains with provider and expiry tracking
- 🌐 **DNS record management** — add/update/remove reverse proxy entries
- 🚫 **Expiry enforcement** — cannot create or update records for expired domains
- 🛡️ **Referential integrity** — cannot delete a domain that still has DNS records
- 📝 **Full audit trail** — every create, update, and delete is logged in a changelog table
- 👤 **User tracking** — `created_by` and `updated_by` fields populated from OS `whoami`
- 🔢 **Versioned migrations** — schema changes tracked like goose/flyway
- 📄 **Beautiful Caddyfile** — organized with banner sections per domain
- 🔄 **Auto-reload** — Caddy is reloaded automatically after every change

---

## Architecture

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│  nurix CLI   │─────▶│  PostgreSQL  │─────▶│  Caddyfile   │
│              │      │  (source of  │      │  (generated) │
│  add/update/ │      │   truth)     │      │              │
│  remove/     │      │              │      │              │
│  search      │      │  - domains   │      │              │
│              │      │  - dns_records│     │              │
│              │      │  - changelog │      │              │
└──────────────┘      └──────────────┘      └──────┬───────┘
                                                   │
       ┌───────────────┐                           │
       │  OS Keyring   │                           ▼
       │  (config)     │                    ┌──────────────┐
       │               │                    │ Caddy reload │
       │  - DB creds   │                    │  (automatic) │
       │  - Caddyfile  │                    └──────────────┘
       │    path       │
       └───────────────┘
```

Every mutation follows the same flow:

1. **Validate** — check domain exists, not expired, etc.
2. **Write** to PostgreSQL
3. **Log** the change to the changelog table
4. **Regenerate** the entire Caddyfile from the database
5. **Reload** Caddy via `systemctl reload caddy`

---

## Prerequisites

- **Go 1.22+** — to build from source
- **PostgreSQL 14+** — running and accessible
- **Caddy 2.x** — installed and managed by systemd
- **Linux / macOS** — for OS keyring support

### Create the database

```bash
sudo -u postgres psql -c "CREATE DATABASE nurix;"
```

Or if you're using a specific user:

```sql
CREATE DATABASE nurix;
GRANT ALL PRIVILEGES ON DATABASE nurix TO your_user;
```

### Open firewall ports for Caddy

If your server uses `ufw`, allow HTTP and HTTPS traffic so Caddy can serve your sites:

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

Verify the rules were added:

```bash
sudo ufw status
```

---

## Installation

### Build from Source

```bash
git clone https://github.com/kashifsb/nurix.git
cd nurix

go mod tidy

go build -ldflags "\
  -X 'github.com/kashifsb/nurix/internal/cli.Version=0.2.6' \
  -X 'github.com/kashifsb/nurix/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)' \
  -X 'github.com/kashifsb/nurix/internal/cli.Commit=$(git rev-parse --short HEAD)'" \
  -o nurix ./cmd/main.go

sudo mv nurix /usr/local/bin/nurix
```

### Using the Makefile

```bash
git clone https://github.com/kashifsb/nurix.git
cd nurix

make install
```

Verify installation:

```bash
nurix --version
# Output: nurix v0.2.6
```

---

## Getting Started

### Step 1 — Configure

Store your database credentials and Caddyfile path securely in the OS keyring:

```bash
nurix set config \
  --caddyfile-path='/etc/caddy/Caddyfile' \
  --dbhost='localhost' \
  --dbport='5432' \
  --dbuser='postgres' \
  --dbpassword='your_secure_password' \
  --dbname='nurix'
```

```
✅ Configuration saved securely to OS keyring

Next step — run database migrations:
  nurix run db-migration
```

> **Note:** Credentials are stored in your OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager). Nothing is written to disk.

### Step 2 — Run Migrations

Initialize the database schema:

```bash
nurix run db-migration
```

```
🔧 Running database migrations...

  ▶️  v1: create domains table ... ✅
  ▶️  v2: create dns_records table ... ✅
  ▶️  v3: create changelog table ... ✅

  3 migration(s) applied.

✅ Database is ready!

Next steps:
  nurix domain add --domain example.com --provider hostinger --expiry 2027-01-01
  nurix dns add --owner app.example.com --target localhost:8080
```

Running it again is safe — already-applied migrations are skipped:

```bash
nurix run db-migration
```

```
🔧 Running database migrations...

  ⏭️  v1: create domains table (already applied)
  ⏭️  v2: create dns_records table (already applied)
  ⏭️  v3: create changelog table (already applied)

  Database is already up to date.
```

### Step 3 — Register Domains

Before creating DNS records, register the base domains you own:

```bash
nurix domain add --domain acrux.ltd --provider hostinger --expiry 2027-06-15
nurix domain add --domain zentix.cloud --provider hostinger --expiry 2026-12-06
nurix domain add --domain sbkashif.com --provider namecheap --expiry 2028-01-01
```

Verify:

```bash
nurix domain search all
```

```
ID   DOMAIN          PROVIDER    EXPIRY       STATUS       CREATED BY   UPDATED BY   CREATED AT            UPDATED AT
--   ------          --------    ------       ------       ----------   ----------   ----------            ----------
1    acrux.ltd       hostinger   2027-06-15   ✅ Active    root         root         2026-03-01 10:00:00   2026-03-01 10:00:00
2    sbkashif.com    namecheap   2028-01-01   ✅ Active    root         root         2026-03-01 10:01:00   2026-03-01 10:01:00
3    zentix.cloud    hostinger   2026-12-06   ✅ Active    root         root         2026-03-01 10:02:00   2026-03-01 10:02:00
```

### Step 4 — Add DNS Records

Now create reverse proxy records under your registered domains:

```bash
nurix dns add --owner app.acrux.ltd --target localhost:8080
nurix dns add --owner api.acrux.ltd --target localhost:3000
nurix dns add --owner zentix.cloud --target localhost:9090
nurix dns add --owner blog.sbkashif.com --target localhost:4000
```

Each command writes to the database, regenerates the Caddyfile, and reloads Caddy:

```
✅ DNS record added: app.acrux.ltd → localhost:8080
📄 Caddyfile written to /etc/caddy/Caddyfile
🔄 Caddy reloaded successfully
```

Verify:

```bash
nurix dns search all
```

```
ID   OWNER               TARGET           DOMAIN         CREATED BY   UPDATED BY   CREATED AT            UPDATED AT
--   -----               ------           ------         ----------   ----------   ----------            ----------
1    api.acrux.ltd       localhost:3000   acrux.ltd      root         root         2026-03-01 10:05:00   2026-03-01 10:05:00
2    app.acrux.ltd       localhost:8080   acrux.ltd      root         root         2026-03-01 10:04:00   2026-03-01 10:04:00
3    blog.sbkashif.com   localhost:4000   sbkashif.com   root         root         2026-03-01 10:07:00   2026-03-01 10:07:00
4    zentix.cloud        localhost:9090   zentix.cloud   root         root         2026-03-01 10:06:00   2026-03-01 10:06:00
```

---

## Commands Reference

### nurix set config

Store configuration values securely in the OS keyring.

```bash
nurix set config \
  --caddyfile-path='/etc/caddy/Caddyfile' \
  --dbhost='localhost' \
  --dbport='5432' \
  --dbuser='postgres' \
  --dbpassword='yourpassword' \
  --dbname='nurix'
```

| Flag               | Required | Description                |
| ------------------ | -------- | -------------------------- |
| `--caddyfile-path` | ✅       | Full path to the Caddyfile |
| `--dbhost`         | ✅       | PostgreSQL host            |
| `--dbport`         | ✅       | PostgreSQL port            |
| `--dbuser`         | ✅       | PostgreSQL username        |
| `--dbpassword`     | ✅       | PostgreSQL password        |
| `--dbname`         | ✅       | PostgreSQL database name   |

---

### nurix run db-migration

Run versioned database migrations.

```bash
nurix run db-migration
```

- Creates tables if they don't exist
- Tracks applied versions in `schema_migrations`
- Skips already-applied migrations
- Each migration runs in a transaction (rolls back on failure)

---

### nurix domain add

Register a new domain.

```bash
nurix domain add --domain example.com --provider hostinger --expiry 2027-06-15
```

| Flag         | Required | Description                                       |
| ------------ | -------- | ------------------------------------------------- |
| `--domain`   | ✅       | Domain name (e.g., `example.com`)                 |
| `--provider` | ❌       | Domain registrar (e.g., `hostinger`, `namecheap`) |
| `--expiry`   | ✅       | Expiry date in `YYYY-MM-DD` format                |

---

### nurix domain update

Update a domain's provider or expiry date.

```bash
nurix domain update --domain example.com --expiry 2029-01-01
nurix domain update --domain example.com --provider cloudflare
nurix domain update --domain example.com --provider cloudflare --expiry 2029-01-01
```

| Flag         | Required | Description                    |
| ------------ | -------- | ------------------------------ |
| `--domain`   | ✅       | Domain to update               |
| `--provider` | ❌       | New provider name              |
| `--expiry`   | ❌       | New expiry date (`YYYY-MM-DD`) |

At least one of `--provider` or `--expiry` must be provided.

---

### nurix domain remove

Remove a registered domain.

```bash
nurix domain remove --domain example.com
```

| Flag       | Required | Description      |
| ---------- | -------- | ---------------- |
| `--domain` | ✅       | Domain to remove |

> ⚠️ **Blocked** if the domain still has DNS records. Remove all DNS records first.

---

### nurix domain search all

List all registered domains.

```bash
nurix domain search all
nurix domain search all --domain example
```

| Flag       | Required | Description                           |
| ---------- | -------- | ------------------------------------- |
| `--domain` | ❌       | Filter by domain name (partial match) |

---

### nurix dns add

Add a new reverse proxy record.

```bash
nurix dns add --owner app.example.com --target localhost:8080
```

| Flag       | Required | Description                                   |
| ---------- | -------- | --------------------------------------------- |
| `--owner`  | ✅       | Domain or subdomain (e.g., `app.example.com`) |
| `--target` | ✅       | Upstream target (e.g., `localhost:8080`)      |

> ⚠️ **Blocked** if the parent domain is not registered or has expired.

---

### nurix dns update

Update the target of an existing DNS record.

```bash
nurix dns update --owner app.example.com --target localhost:8443
```

| Flag       | Required | Description                |
| ---------- | -------- | -------------------------- |
| `--owner`  | ✅       | Domain/subdomain to update |
| `--target` | ✅       | New upstream target        |

> ⚠️ **Blocked** if the parent domain has expired. You can still remove the record.

---

### nurix dns remove

Remove a DNS record.

```bash
nurix dns remove --owner app.example.com
```

| Flag      | Required | Description                |
| --------- | -------- | -------------------------- |
| `--owner` | ✅       | Domain/subdomain to remove |

> ✅ Works even if the parent domain is expired — you can always clean up.

---

### nurix dns search all

List all DNS records.

```bash
nurix dns search all
nurix dns search all --owner example.com
nurix dns search all --target localhost:8080
nurix dns search all --owner example.com --target localhost:8080
```

| Flag       | Required | Description                      |
| ---------- | -------- | -------------------------------- |
| `--owner`  | ❌       | Filter by owner (partial match)  |
| `--target` | ❌       | Filter by target (partial match) |

---

### nurix version

Display version information.

```bash
nurix version
```

```
nurix v0.2.6
  Build date: 2026-03-01T12:00:00Z
  Commit:     a1b2c3d
```

Short form:

```bash
nurix --version
```

```
nurix v0.2.6
```

---

### nurix help

Display help for any command.

```bash
nurix help
nurix --help
nurix domain --help
nurix domain add --help
nurix dns --help
nurix dns search all --help
```

---

## Generated Caddyfile

Nurix generates a clean, organized Caddyfile with banner sections per domain:

```
# Auto-generated by nurix CLI — DO NOT EDIT MANUALLY

##############################
########## SBKASHIF.COM ##########
##############################

blog.sbkashif.com {
	reverse_proxy localhost:4000
}


##############################
########## ZENTIX.CLOUD ##########
##############################

zentix.cloud {
	reverse_proxy localhost:9090
}
```

Domains with no DNS records still appear as empty sections, keeping the structure ready for future records.

---

## Database Schema

### domains

| Column       | Type                           | Description                         |
| ------------ | ------------------------------ | ----------------------------------- |
| `id`         | `SERIAL PRIMARY KEY`           | Auto-incrementing ID                |
| `domain`     | `VARCHAR(255) UNIQUE NOT NULL` | Domain name                         |
| `provider`   | `VARCHAR(255)`                 | Domain registrar                    |
| `expiry`     | `DATE NOT NULL`                | Expiry date                         |
| `created_by` | `VARCHAR(255)`                 | OS user who created the record      |
| `updated_by` | `VARCHAR(255)`                 | OS user who last updated the record |
| `created_at` | `TIMESTAMPTZ`                  | Creation timestamp                  |
| `updated_at` | `TIMESTAMPTZ`                  | Last update timestamp               |

### dns_records

| Column       | Type                                      | Description                         |
| ------------ | ----------------------------------------- | ----------------------------------- |
| `id`         | `SERIAL PRIMARY KEY`                      | Auto-incrementing ID                |
| `owner`      | `VARCHAR(255) UNIQUE NOT NULL`            | Domain/subdomain                    |
| `target`     | `VARCHAR(255) NOT NULL`                   | Upstream target                     |
| `domain_id`  | `INTEGER NOT NULL REFERENCES domains(id)` | Parent domain FK                    |
| `created_by` | `VARCHAR(255)`                            | OS user who created the record      |
| `updated_by` | `VARCHAR(255)`                            | OS user who last updated the record |
| `created_at` | `TIMESTAMPTZ`                             | Creation timestamp                  |
| `updated_at` | `TIMESTAMPTZ`                             | Last update timestamp               |

### changelog

| Column        | Type                   | Description                     |
| ------------- | ---------------------- | ------------------------------- |
| `id`          | `SERIAL PRIMARY KEY`   | Auto-incrementing ID            |
| `entity_type` | `VARCHAR(50) NOT NULL` | `domain` or `dns_record`        |
| `entity_id`   | `INTEGER NOT NULL`     | ID of the affected record       |
| `action`      | `VARCHAR(20) NOT NULL` | `CREATE`, `UPDATE`, or `DELETE` |
| `field_name`  | `VARCHAR(255)`         | Which field changed             |
| `old_value`   | `TEXT`                 | Previous value                  |
| `new_value`   | `TEXT`                 | New value                       |
| `changed_by`  | `VARCHAR(255)`         | OS user who made the change     |
| `changed_at`  | `TIMESTAMPTZ`          | When the change occurred        |

### schema_migrations

| Column        | Type                  | Description                    |
| ------------- | --------------------- | ------------------------------ |
| `version`     | `INTEGER PRIMARY KEY` | Migration version number       |
| `description` | `VARCHAR(255)`        | Human-readable description     |
| `applied_at`  | `TIMESTAMPTZ`         | When the migration was applied |

---

## Validation Rules

| Scenario                                  | Result                                                 |
| ----------------------------------------- | ------------------------------------------------------ |
| `dns add` for unregistered domain         | ❌ Blocked — register the domain first                 |
| `dns add` for expired domain              | ❌ Blocked — shows expiry date                         |
| `dns update` for expired domain           | ❌ Blocked — suggests removal instead                  |
| `dns remove` for expired domain           | ✅ Allowed — you can always clean up                   |
| `domain remove` with existing DNS records | ❌ Blocked — lists records to clean up first           |
| `domain remove` with no DNS records       | ✅ Allowed                                             |
| Any command before `set config`           | ❌ Blocked — shows setup instructions                  |
| Any DB command before `run db-migration`  | ❌ Fails with DB error — user guided to run migrations |
| `dns add` with duplicate owner            | ❌ Blocked — owner must be unique                      |
| `domain add` with duplicate domain        | ❌ Blocked — domain must be unique                     |

---

## Audit Trail

Every change is recorded in the `changelog` table. You can query it directly:

```sql
-- See all changes
SELECT * FROM changelog ORDER BY changed_at DESC;

-- See changes for a specific domain
SELECT * FROM changelog
WHERE entity_type = 'domain' AND entity_id = 1
ORDER BY changed_at DESC;

-- See all DNS record changes by a specific user
SELECT * FROM changelog
WHERE entity_type = 'dns_record' AND changed_by = 'root'
ORDER BY changed_at DESC;

-- See what changed today
SELECT * FROM changelog
WHERE changed_at >= CURRENT_DATE
ORDER BY changed_at DESC;
```

Example changelog entries after creating and then updating a DNS record:

```
id | entity_type | entity_id | action | field_name | old_value      | new_value      | changed_by | changed_at
---|-------------|-----------|--------|------------|----------------|----------------|------------|--------------------
3  | dns_record  | 1         | UPDATE | target     | localhost:8080 | localhost:8443 | root       | 2026-03-01 10:15:00
2  | dns_record  | 1         | CREATE | target     |                | localhost:8080 | root       | 2026-03-01 10:10:00
1  | dns_record  | 1         | CREATE | owner      |                | app.acrux.ltd  | root       | 2026-03-01 10:10:00
```

---

## Security

### Credential Storage

Configuration (including database password) is stored in your operating system's native credential manager:

| OS      | Backend                                  |
| ------- | ---------------------------------------- |
| macOS   | Keychain                                 |
| Linux   | Secret Service (GNOME Keyring / KWallet) |
| Windows | Credential Manager                       |

**No configuration files are written to disk.** The credentials exist only in the OS keyring.

### Database

- The `dns_records` table uses a foreign key to `domains`, ensuring referential integrity
- All mutations are validated before execution
- The changelog provides a complete audit trail

### Caddyfile

- The Caddyfile is overwritten entirely on every change (no partial edits that could cause drift)
- File permissions are set to `0644`

---

## Project Structure

```
nurix/
├── cmd/
│   └── main.go                  # Entrypoint
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root command, subcommand registration
│   │   ├── version.go           # nurix version / --version
│   │   ├── set_config.go        # nurix set config
│   │   ├── run_migration.go     # nurix run db-migration
│   │   ├── domain_add.go        # nurix domain add
│   │   ├── domain_update.go     # nurix domain update
│   │   ├── domain_remove.go     # nurix domain remove
│   │   ├── domain_search.go     # nurix domain search all
│   │   ├── dns_add.go           # nurix dns add
│   │   ├── dns_update.go        # nurix dns update
│   │   ├── dns_remove.go        # nurix dns remove
│   │   └── dns_search.go        # nurix dns search all
│   ├── store/
│   │   ├── connection.go        # PostgreSQL connection
│   │   ├── migration.go         # Versioned schema migrations
│   │   ├── domain.go            # Domain CRUD operations
│   │   ├── dns.go               # DNS record CRUD operations
│   │   └── changelog.go         # Audit trail logging
│   ├── caddy/
│   │   └── sync.go              # Caddyfile generation + Caddy reload
│   └── vault/
│       └── vault.go             # OS keyring credential storage
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## Development

### Clone and build

```bash
git clone https://github.com/kashifsb/nurix.git
cd nurix
go mod tidy
make build
```

### Run locally without installing

```bash
./nurix --help
./nurix set config \
  --caddyfile-path='/tmp/Caddyfile' \
  --dbhost='localhost' \
  --dbport='5432' \
  --dbuser='postgres' \
  --dbpassword='postgres' \
  --dbname='nurix'
./nurix run db-migration
```

### Adding a new migration

Edit `internal/store/migration.go` and add a new entry to the `migrations` slice:

```go
{
    Version:     4,
    Description: "add notes column to domains",
    SQL:         `ALTER TABLE domains ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';`,
},
```

Then run:

```bash
nurix run db-migration
```

Only the new migration will be applied.

### Run tests

```bash
go test ./...
```

---

## Troubleshooting

### "nurix is not configured yet"

You haven't run `nurix set config` yet. See [Step 1](#step-1--configure).

### "failed to connect to database"

- Verify PostgreSQL is running: `systemctl status postgresql`
- Verify the database exists: `psql -l | grep nurix`
- Check your credentials: re-run `nurix set config` with correct values

### "no registered domain found for 'app.example.com'"

You need to register the base domain first:

```bash
nurix domain add --domain example.com --provider hostinger --expiry 2027-01-01
```

### "domain 'example.com' expired on 2025-01-01"

The domain's expiry date has passed. You can:

- Update the expiry: `nurix domain update --domain example.com --expiry 2028-01-01`
- Remove stale DNS records: `nurix dns remove --owner app.example.com`

### "cannot remove domain — it still has X DNS record(s)"

Remove all DNS records under the domain first:

```bash
nurix dns search all --owner example.com    # find all records
nurix dns remove --owner app.example.com    # remove each one
nurix domain remove --domain example.com    # now this works
```

### "Caddyfile written but reload failed"

- Verify Caddy is installed: `caddy version`
- Verify Caddy is managed by systemd: `systemctl status caddy`
- Check Caddyfile syntax: `caddy validate --config /etc/caddy/Caddyfile`
- Check permissions: the nurix binary needs permission to write the Caddyfile and reload Caddy (run with `sudo` if needed)

### Keyring issues on headless Linux servers

If your Linux server doesn't have a desktop environment, the Secret Service backend (`go-keyring`) may not be available.
In this case, Nurix automatically falls back to an encrypted configuration file stored at `~/.nurix/config.enc`.

The file is encrypted using AES-GCM. The encryption key is derived securely from your machine's unique identifiers (hostname, user ID, and `/etc/machine-id`), ensuring that the credentials cannot be decrypted if the file is copied to another machine.

No extra setup is required!

---

## License

MIT License. See [LICENSE](LICENSE) for details.
