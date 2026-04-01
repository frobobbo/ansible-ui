# Ansible UI

A self-hosted web interface for running Ansible playbooks via SSH. Define forms with custom fields, assign them to hosts and playbooks, and let your team trigger automated tasks from a browser — no CLI required.

## Features

- **Role-based access** — Admin, Editor, and Viewer roles with granular permissions
- **Quick Actions** — Pin frequently-used forms to the dashboard as one-click cards with optional custom images
- **Form builder** — Create forms with text, number, boolean, and select fields that map to Ansible extra-vars
- **Scheduled runs** — Attach a cron expression to any form; it runs automatically on schedule using field default values, surviving server restarts
- **Vault support** — Store encrypted vault passwords; automatically passed as `--vault-password-file` at run time
- **SSH Certificates** — Upload and manage SSH private keys; associate them with Hosts for automatic injection at run time
- **Hosts inventory** — Manage Ansible target hosts (name, address, per-host vars, SSH cert) separately from Job Runners
- **Host Groups** — Organize hosts into named groups for multi-host playbook targeting
- **Job Runners** — Execution servers that run `ansible-playbook` over SSH (classic runner)
- **Execution Environments** — Run playbooks inside a container image on Kubernetes for reproducible, isolated execution
- **EE Editor** — In-app editor to manage EE package files (`execution-environment.yml`, `requirements.yml`, etc.) and push changes to GitHub, triggering an automated rebuild
- **Live run output** — Stream stdout/stderr from `ansible-playbook` in real time
- **Run history** — Browse past runs and replay them with a single click
- **Audit log** — Record of all create/update/delete actions with user attribution
- **Responsive UI** — Mobile-friendly sidebar with hamburger navigation
- **Single binary** — Go backend serves the pre-built SvelteKit frontend; no external runtime dependencies

## Roles

| Role   | Dashboard | Quick Actions | Forms | Run History | Infrastructure/Secrets/Users |
|--------|-----------|---------------|-------|-------------|------------------------------|
| Admin  | ✓         | ✓             | ✓     | ✓           | ✓                            |
| Editor | ✓         | ✓             | ✓     | ✓           | —                            |
| Viewer | ✓         | ✓             | —     | —           | —                            |

## Hosts vs Job Runners

These are two distinct concepts:

| Concept | Purpose |
|---|---|
| **Host** | The Ansible target — the machine a playbook runs *against* (`hosts:` in the playbook). Stores name, IP/hostname, per-host vars, and an optional SSH cert. |
| **Job Runner** | The execution server — the machine that *runs* `ansible-playbook` over SSH (classic runner only). Not needed when using Execution Environments. |

## SSH Certificates

SSH private keys are stored as **SSH Certs** (under the Secrets nav group) and uploaded as files. You can attach a cert to any Host; the runner will automatically:

1. Mount the key into the job environment
2. Set `ansible_ssh_private_key_file` in the generated inventory

Keys are never exposed through the API after upload.

## Execution Environments

An Execution Environment (EE) is a container image that provides a fully self-contained `ansible-playbook` runtime. When a form targets an EE runner:

1. A Kubernetes Job is created with the specified image
2. Playbook, inventory, vault files, and SSH cert are injected via ConfigMap/Secret volumes
3. `ansible-playbook` runs inside the container; output streams back to the UI
4. The Job is cleaned up after completion

The default base image is `ghcr.io/ansible-community/community-ee-base:latest`. A custom image (`ghcr.io/frobobbo/ansible-ee:latest`) is built automatically by the included GitHub Actions workflow whenever EE package files change.

### EE Editor

Admins can manage EE package files directly from the UI at **Infrastructure → EE Editor**:

| File | Purpose |
|---|---|
| `execution-environment.yml` | `ansible-builder` definition — base image, dependency references |
| `requirements.yml` | Ansible Galaxy collections |
| `requirements.txt` | Python pip packages |
| `bindep.txt` | System (RPM/DEB) packages |

Changes are committed to GitHub via the Contents API, which triggers the `build-ee.yml` GitHub Actions workflow to rebuild and push the image.

**Required configuration** (Helm or env vars):

| Variable | Description |
|---|---|
| `GITHUB_TOKEN` | Fine-grained PAT with **Contents: Read & Write** on the repository |
| `GITHUB_REPO` | Repository in `owner/repo` format, e.g. `frobobbo/ansible-ui` |
| `GITHUB_BRANCH` | Branch to commit to (default: `main`) |

### Building the EE manually

```bash
pip install ansible-builder
cd execution-environment
ansible-builder build -t my-ee:latest
```

## Scheduling

Any form can be set to run automatically on a cron schedule:

1. Open a form in the editor and expand the **Scheduling** card.
2. Enable **Run on a schedule** and enter a cron expression.
3. Save — the next run time is shown immediately.

Supported formats:

| Expression | Meaning |
|---|---|
| `0 2 * * *` | Every day at 02:00 UTC |
| `*/15 * * * *` | Every 15 minutes |
| `0 9 * * 1` | Every Monday at 09:00 UTC |
| `@hourly` | Once per hour |
| `@daily` | Once per day at midnight UTC |
| `@weekly` | Once per week on Sunday UTC |

Scheduled runs use the **field default values** defined on the form. All times are UTC.

## Tech Stack

- **Backend**: Go + Gin, JWT (HS256), `modernc.org/sqlite` (no CGO required)
- **Frontend**: SvelteKit 5 (runes) + TypeScript, built as a static SPA
- **SSH**: `golang.org/x/crypto/ssh`
- **Scheduler**: `github.com/robfig/cron/v3` — standard 5-field cron + `@hourly`/`@daily`/`@weekly`
- **Kubernetes runner**: `k8s.io/client-go` — Jobs, ConfigMaps, Secrets, Pod log streaming
- **Database**: SQLite in WAL mode at `./data/ansible.db`

## Getting Started

### Docker (recommended)

```bash
docker compose up -d
```

The app listens on port `8080`. Open [http://localhost:8080](http://localhost:8080) and log in with:

- **Username**: `admin`
- **Password**: `admin`

### Kubernetes (Helm)

**Add the chart repository:**

```bash
helm repo add ansible-ui https://frobobbo.github.io/ansible-ui
helm repo update
```

**Install with default settings:**

```bash
helm install ansible-ui ansible-ui/ansible-ui \
  --set secret.jwtSecret=$(openssl rand -hex 32) \
  --set secret.adminPassword=yourpassword
```

**Install with Ingress:**

```bash
helm install ansible-ui ansible-ui/ansible-ui \
  --set secret.jwtSecret=$(openssl rand -hex 32) \
  --set secret.adminPassword=yourpassword \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set "ingress.hosts[0].host=ansible-ui.yourdomain.com" \
  --set "ingress.hosts[0].paths[0].path=/" \
  --set "ingress.hosts[0].paths[0].pathType=Prefix"
```

**Install with EE Editor (GitHub integration):**

```bash
helm install ansible-ui ansible-ui/ansible-ui \
  --set secret.jwtSecret=$(openssl rand -hex 32) \
  --set secret.adminPassword=yourpassword \
  --set eeEditor.githubToken=ghp_yourtoken \
  --set eeEditor.githubRepo=yourorg/yourrepo \
  --set eeEditor.githubBranch=main
```

`GITHUB_TOKEN` is stored in the same Kubernetes Secret as `JWT_SECRET` and `ADMIN_PASSWORD` — it is never exposed as a plain environment value.

**Access without Ingress** (port-forward):

```bash
kubectl port-forward svc/ansible-ui 8080:80
# Open http://localhost:8080
```

**Key chart values:**

| Value | Default | Description |
|---|---|---|
| `secret.jwtSecret` | `change-me-...` | JWT signing key — always override |
| `secret.adminPassword` | `admin` | Initial admin password |
| `secret.existingSecret` | `""` | Use a pre-existing Secret instead |
| `persistence.size` | `2Gi` | PVC size for database + uploads |
| `persistence.storageClass` | `""` | Storage class (cluster default if empty) |
| `persistence.existingClaim` | `""` | Use a pre-existing PVC instead |
| `ingress.enabled` | `false` | Enable Ingress resource |
| `service.type` | `ClusterIP` | `ClusterIP`, `NodePort`, or `LoadBalancer` |
| `eeEditor.githubToken` | `""` | GitHub PAT for EE Editor commits (stored in Secret) |
| `eeEditor.githubRepo` | `""` | Repository for EE files, e.g. `owner/repo` |
| `eeEditor.githubBranch` | `main` | Branch to commit EE changes to |
| `rbac.create` | `true` | Create Role + RoleBinding for K8s Job management |

The full values reference is in [`helm/ansible-ui/values.yaml`](helm/ansible-ui/values.yaml).

> **Note:** The chart uses SQLite with a `ReadWriteOnce` volume and a `Recreate` deployment strategy. Keep `replicaCount: 1`.

### Build from source

**Prerequisites**: Go 1.22+, Node.js 20+

```bash
# Build frontend
cd frontend
npm install
npm run build
cd ..

# Build and run backend
cd backend
go build -o ../ansible-frontend .
cd ..
./ansible-frontend
```

## Configuration

Set environment variables before starting the server:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `JWT_SECRET` | `change-me` | Secret key for signing JWTs — change in production |
| `ADMIN_PASSWORD` | `admin` | Initial password for the built-in admin account |
| `GITHUB_TOKEN` | — | GitHub PAT for EE Editor (fine-grained: Contents read/write) |
| `GITHUB_REPO` | — | Repository for EE files, e.g. `owner/repo` |
| `GITHUB_BRANCH` | `main` | Branch to commit EE changes to |

## Docker Compose

```yaml
services:
  ansible-ui:
    image: ghcr.io/frobobbo/ansible-ui:latest
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    environment:
      JWT_SECRET: your-secret-here
      ADMIN_PASSWORD: your-admin-password
      # Optional — enables the in-app EE Editor
      # GITHUB_TOKEN: ghp_yourtoken
      # GITHUB_REPO: yourorg/yourrepo
      # GITHUB_BRANCH: main
    restart: unless-stopped
```

Data (database, uploaded playbooks, vault files, form images) is persisted in the `./data` volume mount.

## Project Structure

```
ansible-frontend/
├── backend/                    Go source (Gin API, SQLite store, runners)
│   ├── internal/
│   │   ├── api/                HTTP handlers (forms, runs, hosts, EE editor, …)
│   │   ├── auth/               JWT middleware
│   │   ├── models/             Shared data types
│   │   ├── runner/
│   │   │   ├── ansible.go      Classic SSH runner (ssh + ansible-playbook)
│   │   │   └── k8s.go          Kubernetes EE runner (Jobs + ConfigMaps)
│   │   ├── scheduler/          Cron scheduler (robfig/cron/v3)
│   │   └── store/              SQLite queries
│   └── main.go
├── frontend/                   SvelteKit 5 source
│   └── src/
│       ├── lib/                API client, stores, types
│       └── routes/             Pages (forms, hosts, runs, ee, ssh-certs, …)
├── execution-environment/      EE build files (managed via EE Editor or git)
│   ├── execution-environment.yml
│   ├── requirements.yml
│   └── requirements.txt
├── .github/workflows/
│   ├── docker.yml              Build + push app image to ghcr.io
│   ├── build-ee.yml            Build + push EE image on execution-environment/** changes
│   └── release-chart.yml       Publish Helm chart to GitHub Pages
├── helm/
│   └── ansible-ui/             Helm chart
├── Dockerfile
└── docker-compose.yml
```

## Development

```bash
# Terminal 1 — backend
cd backend && go run .

# Terminal 2 — frontend dev server (proxies /api → localhost:8080)
cd frontend && npm run dev
```

The frontend dev server runs on [http://localhost:5173](http://localhost:5173) and proxies API requests to the Go backend on port 8080.

## License

MIT
