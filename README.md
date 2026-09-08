# Copyarr

Copyarr is a small persistent transfer queue for automated copy/move jobs. It is designed to run as an Unraid container today and expose a simple HTTP API that can later be surfaced inside Nexus.

## MVP goals

- Source/destination abstraction through rclone (local, FTP, SFTP and later cloud remotes).
- Persistent SQLite state; destination folders do not need to retain history.
- `ignore_existing` bootstrap so a new rule can start from "only things that appear from now on".
- Stability-based readiness fallback.
- Optional rTorrent XML-RPC readiness: only paths belonging to completed torrents are queued.
- Persistent grouped job queue. Restarting the container does not erase state.
- One completed rTorrent payload becomes one Copyarr job, with all files verified as a manifest.
- Copy into a hidden `.copyarr-staging/<job-id>/...` area, verify exact byte sizes, then commit to the final destination.
- `move` means copy + verify + commit before deleting the source.
- Managed 14-day cleanup only touches destinations Copyarr itself committed and whose size still matches.
- Small HTTP API for health/state/scan triggers.

## Current status

The transfer engine now uses persistent jobs rather than a file-only queue. Rules are still JSON-configured and the worker is intentionally single-transfer for predictable seedbox behaviour. Dynamic rule editing, Nexus registration, checksums and richer retry policy remain later iterations.

## Quick start

1. Create an rclone config containing your source remote(s). For FTP this can be done with `rclone config`.
2. Copy `config.example.json` to `/mnt/user/appdata/copyarr/config/config.json` and edit it.
3. Do **not** reuse credentials that have been pasted into chat/logs; rotate them first.
4. Run `ghcr.io/swamp2k/copyarr:latest` with the example compose file or an Unraid template.

Copyarr publishes a fresh `:latest` image from `main` after CI succeeds. Git tags such as `v0.2.0` also publish matching version tags. Unraid can therefore detect updates by comparing the registry digest instead of relying on locally-built `copyarr:local` images.

The repository includes an Unraid Docker template at `unraid/copyarr.xml`. It uses `ghcr.io/swamp2k/copyarr:latest`, so Unraid can detect updates by image digest.

The example mounts:

- `/mnt/user/appdata/copyarr-data/data` → `/data`
- `/mnt/user/appdata/copyarr-data/config` → `/config`
- `/mnt/user/media/downloads` → `/downloads`

## rTorrent

Copyarr talks to an HTTP XML-RPC endpoint such as `/RPC2`. Do not expose raw rTorrent SCGI publicly; rTorrent expects access control to be provided by the web server/reverse proxy in front of it.

When enabled, Copyarr calls `d.multicall2` for hash, name, completion state and base path. Completed torrent base paths are mapped relative to `source_base_path` and used as a readiness gate. If `required` is false and the RPC endpoint is unavailable, the rule falls back to the stability timer.

## API

- `GET /health`
- `GET /api/status` — active transfer, queue depth/bytes, job counts, progress, speed and ETA.
- `GET /api/rules`
- `GET /api/jobs?limit=100`
- `GET /api/objects?limit=200`
- `POST /api/scan`

Default listen address: `:8686`.

## State model

Objects typically move through:

`discovered -> queued -> copying -> done`

Jobs independently move through:

`queued -> copying -> done`

with `retry_wait` on transfer failure, `ignored` for initial bootstrap objects, `superseded` for an older generation of a still-growing upload, and `cleaned` after managed retention cleanup.

Object identity includes relative path, exact size and modification time. A destination can therefore be moved away by Sonarr/Tdarr without Copyarr "forgetting" that the source generation was already handled.

## Safety decisions

Copyarr does not compare source and destination directories to decide what is new. It also does not recursively delete arbitrary old files from a destination. Cleanup only considers files recorded as successfully committed by Copyarr and skips files whose current size differs from the recorded size.
