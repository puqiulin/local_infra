# Jellyfin — Design

Date: 2026-06-15
Status: Approved (pending spec review)

## Goal

Add a Jellyfin media server to the existing `local-infra` Docker Compose stack,
serving media from `/Users/apple/work/videos`, and wire it into the existing
observability stack (Prometheus, Grafana, blackbox) the same way every other
service is wired.

## Conventions followed

The stack uses upstream official images, a single `local-infra` bridge network,
`restart: unless-stopped`, named volumes with `driver: local`, host port
mappings, one Prometheus scrape job per service, a dedicated exporter per
stateful service (postgres/redis/kafka), Grafana dashboards provisioned from
`grafana/provisioning/dashboards/json/`, and public domains health-checked via
the blackbox `http_2xx` probe.

## Decisions

- **Image:** `jellyfin/jellyfin:latest` (official upstream, matches convention —
  not the linuxserver.io image, so no PUID/PGID env vars).
- **Media:** bind-mount `/Users/apple/work/videos` read-only at `/media`
  (contains `animation/`, `drama/`, `movie/`).
- **Metrics:** dedicated **`rebelcore/jellyfin-exporter`** (port `9594`), chosen
  over Jellyfin's native `/metrics`. Native metrics are off by default, require a
  manual `system.xml` edit after first boot in the current release, and expose
  only sparse runtime/item-count data. The exporter exposes rich metrics (active
  streams, transcodes, sessions, users, library counts) and matches the
  one-exporter-per-service pattern.
- **Exposure:** public domain `jellyfin.sprite3366.com`, health-checked via
  blackbox. TLS/reverse-proxy termination is external to this repo (same as the
  other public domains) and is out of scope here.
- **Host port:** `8096` (Jellyfin default; verified free).

## Components

### 1. `jellyfin` service (docker-compose.yml)

```yaml
jellyfin:
  image: jellyfin/jellyfin:latest
  container_name: jellyfin
  restart: unless-stopped
  ports:
    - "8096:8096"
  environment:
    TZ: ${TZ:-Asia/Shanghai}
    JELLYFIN_PublishedServerUrl: https://jellyfin.sprite3366.com
  volumes:
    - jellyfin_config:/config
    - jellyfin_cache:/cache
    - /Users/apple/work/videos:/media:ro
  networks:
    - local-infra
```

### 2. `jellyfin-exporter` service (docker-compose.yml)

```yaml
jellyfin-exporter:
  image: rebelcore/jellyfin-exporter:latest
  container_name: jellyfin_exporter
  restart: unless-stopped
  command:
    - "--jellyfin.address=http://jellyfin:8096"
    - "--jellyfin.token=${JELLYFIN_TOKEN}"
  ports:
    - "9594:9594"
  depends_on:
    - jellyfin
  networks:
    - local-infra
```

### 3. New named volumes (docker-compose.yml `volumes:`)

```yaml
jellyfin_config:
  driver: local
jellyfin_cache:
  driver: local
```

(No volume for media — it is a host bind mount.)

### 4. Prometheus (`prometheus/prometheus.yml`)

- New scrape job:

  ```yaml
  - job_name: jellyfin_exporter
    static_configs:
      - targets: ["jellyfin-exporter:9594"]
  ```

- Add `https://jellyfin.sprite3366.com` to the existing `blackbox` job's
  `static_configs.targets` list.

### 5. Grafana dashboard

- New file `grafana/provisioning/dashboards/json/jellyfin-overview.json`,
  picked up automatically by the existing file provider.
- Follows the existing dashboards: dashboard uid `jellyfin-overview`, a
  `${datasource}` Prometheus template variable, panels driven by
  `jellyfin-exporter` metrics (up/scrape status, sessions, active streams,
  transcodes, library item counts, users).

### 6. `.env` (gitignored — documented, not committed)

Add:

- `JELLYFIN_TOKEN=<api key created during Jellyfin setup>`
- optionally `TZ=Asia/Shanghai`

## Data flow

Browser/clients → `:8096` (or `jellyfin.sprite3366.com` via external proxy) →
Jellyfin reads media from `/media` (host `/Users/apple/work/videos`).
`jellyfin-exporter` polls the Jellyfin HTTP API (`jellyfin:8096`) with the API
token and exposes `:9594/metrics`. Prometheus scrapes `jellyfin-exporter:9594`
and probes the public URL via blackbox. Grafana renders the provisioned
dashboard from the Prometheus datasource.

## Error handling / edge cases

- **Exporter before token exists:** `jellyfin-exporter` will fail to scrape until
  `JELLYFIN_TOKEN` is set and Jellyfin setup is complete. Acceptable — it is part
  of the documented one-time bootstrap, and `restart: unless-stopped` recovers it
  after the token is provided.
- **First-run wizard:** Jellyfin requires an interactive first-run setup that
  cannot be automated in the official image; captured as a manual step below.
- **Media read-only:** mounted `:ro` so Jellyfin cannot modify source files.

## Post-deploy manual steps (cannot be automated)

1. `docker compose up -d jellyfin`
2. Open `http://localhost:8096`, complete the setup wizard; add a library
   pointing at `/media` (with `animation`, `drama`, `movie` subfolders).
3. Dashboard → Administration → API Keys → create a key.
4. Put the key in `.env` as `JELLYFIN_TOKEN`.
5. `docker compose up -d jellyfin-exporter`
6. `docker compose up -d prometheus grafana` (or reload Prometheus:
   `curl -X POST http://localhost:9090/-/reload`) to pick up the new scrape job,
   blackbox target, and dashboard.
7. In the external reverse proxy, point `jellyfin.sprite3366.com` → host `:8096`.

## Verification

- `docker compose config` parses without error.
- `docker compose up -d jellyfin jellyfin-exporter` → both containers healthy.
- `http://localhost:8096` serves the Jellyfin UI.
- After token setup: `http://localhost:9594/metrics` returns Jellyfin metrics.
- Prometheus `Targets` page shows `jellyfin_exporter` UP and the new blackbox
  target probed.
- Grafana shows the `jellyfin-overview` dashboard with live data.

## Out of scope

- Reverse proxy / TLS configuration (external to this repo).
- Hardware-accelerated transcoding (no GPU passthrough on Docker for macOS;
  transcoding will be CPU/software only).
- Loki/promtail log wiring (Jellyfin logs are already captured by the existing
  docker-wide promtail config).
