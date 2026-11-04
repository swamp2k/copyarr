# Repository instructions

- Read root NOW.md at the start of every session.
- Update root NOW.md before ending every session, and keep it current as work progresses. Use Doing, State, Next step, Blocked by, and Updated (YYYY-MM-DD). Preserve relevant content, base entries on verified findings, and explicitly mark uncertainty.
- Whenever either root AGENTS.md or CLAUDE.md changes, update both identically in the same change. Preserve useful existing instructions. Leave nested instruction files unchanged.
- Read README.md and HANDOVER.md when present before implementation; follow applicable nested instructions.
- Preserve transfer safety: copy to staging, verify, and commit before deleting move sources; cleanup must remain limited to recorded managed destinations.
- For code changes, follow the CI checks in .github/workflows/ci.yml: Go 1.23, go mod tidy, go test ./..., go vet ./..., and docker build -t copyarr:ci . Report checks that could not run.
