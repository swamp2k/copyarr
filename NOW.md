# NOW

Doing: Re-orientation (2026-11-04) completed; nothing changed on main since 2026-11-02. Previously: root instructions and bookmark published on branch docs/session-bookmark-2026-11-02 in PR #1 (https://github.com/swamp2k/copyarr/pull/1). Awaiting user direction before implementation.

State:
- Reviewed main at d78eb4dafae9c04d4ad39ac8c9b51cc3d99e9d96, README, CI, entry point, engine, API, Privateering pusher/settings, existing snapshot test, recent commits, and open PRs.
- Root AGENTS.md, CLAUDE.md, HANDOVER.md, and NOW.md were absent. Created identical root instruction files and this bookmark; fetched contents verified identical. Recursive repository listing contains no nested instruction or handover files.
- Go 1.23 service with SQLite state, serial rclone transfers, staged size verification, retries, job controls, dashboard, remote/job configuration, filtering, and managed retention cleanup.
- Latest commits (2026-10-07): optional Nexus Privateering snapshot push, then web UI/SQLite settings replacing environment/config settings. Earlier recent work covered responsive dashboard and log retention.
- No open pull requests existed at inspection (before this documentation PR).
- CI runs go mod tidy, go test ./..., go vet ./..., and Docker build on pushes/PRs. Publishing to GHCR happens only after successful CI on main or version tags.
- Static finding: Pusher.lastPush is accessed without synchronization by the background schedule and HTTP-triggered PushNow. Race-detector reproduction is pending; runtime impact is unverified. No direct pusher/settings tests exist in internal/privateering; engine snapshot coverage exists.
- README still describes Nexus registration, accounts/auth, and checksums as later work. Deployment health and current test results are unverified.
- Verified PR #1 changes only AGENTS.md, CLAUDE.md, and NOW.md; no implementation changes. PR #1 CI passed (verified 2026-11-04). Root instruction files still absent on main until PR #1 merges.
- 2026-11-04: re-confirmed lastPush is a plain time.Time field read/written in maybePush and PushNow with no mutex (pusher.go:60,99,102,118). Data race still unreproduced at runtime.

Next step: DONE

Blocked by: Further implementation awaits user direction as requested. PR #1 needs merging to land instructions on main. Local clone now works (local checkout, Go installed); local Go/Docker checks not yet run.

Updated: 2026-11-04
