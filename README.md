# Ansible UI

A self-hosted web interface for running Ansible playbooks via SSH. Define forms with custom fields, assign them to servers and playbooks, and let your team trigger automated tasks from a browser — no CLI required.

## Features

- **Role-based access** — Admin, Editor, and Viewer roles with granular permissions
- **Quick Actions** — Pin frequently-used forms to the dashboard as one-click cards with optional custom images
- **Form builder** — Create forms with text, number, boolean, and select fields that map to Ansible extra-vars
- **Vault support** — Store encrypted vault passwords; automatically passed as `--vault-password-file` at run time
- **Live run output** — Stream stdout/stderr from `ansible-playbook` in real time
- **Run history** — Browse past runs and replay them with a single click
- **Single binary** — Go backend serves the pre-built SvelteKit frontend; no external runtime dependencies

## Roles

| Role   | Dashboard | Quick Actions | Forms | Run History | Servers/Playbooks/Vaults/Users |
|--------|-----------|---------------|-------|-------------|-------------------------------|
| Admin  | ✓         | ✓             | ✓     | ✓           | ✓                             |
| Editor | ✓         | ✓             | ✓     | ✓           | —                             |
| Viewer | ✓         | ✓             | —     | —           | —                             |

## Tech Stack

- **Backend**: Go + Gin, JWT (HS256), `modernc.org/sqlite` (no CGO required)
- **Frontend**: SvelteKit 5 (runes) + TypeScript, built as a static SPA
- **SSH**: `golang.org/x/crypto/ssh`
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

| Variable         | Default     | Description                                      |
|------------------|-------------|--------------------------------------------------|
| `PORT`           | `8080`      | HTTP listen port                                 |
| `JWT_SECRET`     | `change-me` | Secret key for signing JWTs — change in production |
| `ADMIN_PASSWORD` | `admin`     | Initial password for the built-in admin account  |

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
    restart: unless-stopped
```

Data (database, uploaded playbooks, vault files, form images) is persisted in the `./data` volume mount.

## Project Structure

```
ansible-frontend/
├── backend/           Go source (Gin API, SQLite store, SSH runner)
│   ├── internal/
│   │   ├── api/       HTTP handlers
│   │   ├── auth/      JWT + middleware
│   │   ├── models/    Shared data types
│   │   ├── runner/    ansible-playbook execution via SSH
│   │   └── store/     SQLite queries
│   └── main.go
├── frontend/          SvelteKit 5 source
│   └── src/
│       ├── lib/       Shared utilities (api client, stores, types)
│       └── routes/    Pages
├── helm/
│   └── ansible-ui/    Helm chart
├── Dockerfile
└── docker-compose.yml
```

## Development

```bash
# Terminal 1 — backend with hot reload
cd backend && go run .

# Terminal 2 — frontend dev server (proxies /api → localhost:8080)
cd frontend && npm run dev
```

The frontend dev server runs on [http://localhost:5173](http://localhost:5173) and proxies API requests to the Go backend on port 8080.

## License

MIT
