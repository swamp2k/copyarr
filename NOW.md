# NOW

Doing: Repository orientation completed; publishing the root instruction files and this session bookmark on branch docs/session-bookmark-2026-11-02 for review. Awaiting user direction before implementation.

State:
- Reviewed main at d78eb4dafae9c04d4ad39ac8c9b51cc3d99e9d96, README, CI, entry point, engine, API, Privateering pusher/settings, existing snapshot test, recent commits, and open PRs.
- Root AGENTS.md, CLAUDE.md, HANDOVER.md, and NOW.md were absent. Creating identical root instruction files and this bookmark. Recursive repository listing contains no nested instruction or handover files.
- Go 1.23 service with SQLite state, serial rclone transfers, staged size verification, retries, job controls, dashboard, remote/job configuration, filtering, and managed retention cleanup.
- Latest commits (2026-10-07): optional Nexus Privateering snapshot push, then web UI/SQLite settings replacing environment/config settings. Earlier recent work covered responsive dashboard and log retention.
- No open pull requests existed at inspection (before this documentation PR).
- CI runs go mod tidy, go test ./..., go vet ./..., and Docker build on pushes/PRs. Publishing to GHCR happens only after successful CI on main or version tags.
- Static finding: Pusher.lastPush is accessed without synchronization by the background schedule and HTTP-triggered PushNow. Race-detector reproduction is pending; runtime impact is unverified. No direct pusher/settings tests exist in internal/privateering; engine snapshot coverage exists.
- README still describes Nexus registration, accounts/auth, and checksums as later work. Deployment health and current test results are unverified.
- Only AGENTS.md, CLAUDE.md, and NOW.md are being changed; no implementation changes.

Next step: After user direction, reproduce concurrent scheduled/manual Nexus pushes with a focused race-detector test; if confirmed, synchronize scheduling state and add regression coverage. Also cover settings token preservation/clearing and URL normalization. Run repository CI checks and go test -race for affected packages.

Blocked by: Further implementation awaits user direction as requested. Local clone failed because the workspace HTTP proxy was unreachable; inspection/publication use the GitHub connector. Local Go/Docker checks have not run; PR CI result is pending.

Updated: 2026-11-02
