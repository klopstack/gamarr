# Gamarr fork → upstream PR split plan

**Fork:** `klopstack/gamarr` branch `dev`  
**Upstream:** `JeremiahM37/gamarr` branch `main` (`f888156` as of 2026-09-08)  
**Inventory date:** 2026-09-08

---

## Execution status (2026-09-08)

| Step | Result |
|------|--------|
| Rebase `dev` onto `upstream/main` | **Done** — resolved README + scheduler conflicts; skipped/dropped upstream-duplicated qBit API-key commits; `vimm.go` matches upstream (0 diff) |
| `go test ./...` on rebased `dev` | **PASS** |
| Force-push `origin/dev` | **Done** — `ef67a03` |
| PR branches pushed | **8/8** — see table below |
| Upstream PRs opened | **8/8** — [#48–#55](https://github.com/JeremiahM37/gamarr/pulls) |

### Branch SHAs (origin)

| Merge order | Branch | SHA | Upstream PR |
|-------------|--------|-----|-------------|
| 1 | `fix/job-invariants-recovery` | `8dbc5d6` | [#48](https://github.com/JeremiahM37/gamarr/pull/48) |
| 2 | `feat/qbit-archive-ops` | `70ae190` | [#49](https://github.com/JeremiahM37/gamarr/pull/49) |
| 3 | `feat/minerva-search` | `c332cd4` | [#50](https://github.com/JeremiahM37/gamarr/pull/50) |
| 4 | `feat/minerva-archive-download` | `1cd276f` | [#52](https://github.com/JeremiahM37/gamarr/pull/52) |
| 5 | `feat/minerva-per-rom-import` | `0c01829` | [#53](https://github.com/JeremiahM37/gamarr/pull/53) |
| 6 | `feat/archive-downloads-ui` | `569c878` | [#54](https://github.com/JeremiahM37/gamarr/pull/54) |
| 7 | `fix/minerva-job-lifecycle` | `e354968` | [#55](https://github.com/JeremiahM37/gamarr/pull/55) |
| 8 | `fix/sabnzbd-inprocess-nzb` | `8690dab` | [#51](https://github.com/JeremiahM37/gamarr/pull/51) |

**Notes:**
- PR branches 3–7 include minimal Minerva source/search wiring so they compile standalone against `upstream/main` (manager references `Sources.Minerva`).
- PR #55 (`fix/minerva-job-lifecycle`) is the integration branch — nearly full `dev` delta minus SAB (PR #51).
- Homelab pin (`klopstack/gamarr#dev`) now tracks rebased history; no homelab-casc changes made.

---

## Executive summary

The `dev` branch is **8 commits** (1 squash-merge of `fix/archive-ui-polish` + 7 fix commits) with **8 uncommitted files** (+322/−32 lines). There is **no merge base** with `upstream/main` — histories diverged when `dev` was created from a standalone merge. Before opening PRs, **rebase `dev` onto latest `upstream/main`** and resolve conflicts (expect heavy overlap in `manager.go`, `vimm.go`, `nzb.go`, `db.go`, `app.js`).

`go test ./...` is **green** on the working tree (including uncommitted changes).

**Homelab:** `homelab-casc/stacks/mediastack/compose.yaml` builds from `https://github.com/klopstack/gamarr.git#dev`. No homelab-specific code lives in the gamarr fork — safe to upstream everything below.

---

## A. Inventory: upstream vs fork

### Already merged upstream (do NOT re-submit)

| Feature | Upstream ref | Notes |
|---------|--------------|-------|
| qBittorrent ≥5.2 Bearer API key | PR #44 (`37c1236`) | Merged from `klopstack/feat/qbittorrent-api-key`. Dev builds *on top* with `AddTorrentOpts`, `SetFilePriority`, `RenameTorrent`. |
| Vimm rate-limit + 404 handling | PR #45/46 (`f888156`) | **Dev is behind.** Fork has older `vimm.go` (missing alias fix, Retry-After doc, gate-hold change). Take upstream during rebase. |
| Import race hardening | PR #42 | Upstream only — may conflict with Minerva per-ROM import in `manager.go`. |
| Vault placeholder / archive hardening | PR #41–43 | Upstream vault-archive logic overlaps Minerva file-priority path. Merge carefully. |
| PC vault archive (one archive per game) | PR #30 | Upstream; fork retains compatible `vaultArchiveEnabled()` paths. |

### Fork-only (47 files, +2822/−449 vs `upstream/main`)

Grouped by feature area:

#### 1. Minerva search indexer
| Path | Status |
|------|--------|
| `internal/search/minerva.go` | **New** (332 lines) — browse-page cache, local title match, shared magnet |
| `internal/search/minerva_test.go` | **New** (272 lines) |
| `internal/sources/defaults.json` | Minerva registry entry + platform path mappings |
| `internal/sources/sources.go`, `load.go`, `sources_test.go` | Driver wiring |
| `internal/search/filter.go`, `filter_test.go` | Minerva result handling |
| `internal/search/scoring.go`, `scoring_test.go` | Minerva score boost |
| `internal/search/sourcename.go`, `sourcename_test.go` | Source label |
| `internal/search/health.go`, `health_test.go` | Health probe |
| `internal/search/registry_flow_test.go` | Registry integration |
| `internal/api/search_indexers_test.go` | API test |
| `internal/api/admin.go` | Minerva health card |
| `internal/api/torznab_wire.go` | Minor wiring |
| `cmd/gamarr/main.go` | 4th search goroutine (`SearchMinerva`) |
| `internal/api/requests.go` | Same in request-search path |
| `README.md` | Minerva docs, platform table column, `MINERVA_URL` |
| `web/static/js/app.js` | `isMinerva` badge (hide seeders/leechers) |

#### 2. Minerva archive download + file selection
| Path | Status |
|------|--------|
| `internal/download/manager.go` | `DownloadTorrent(..., selectFiles)`, `selectArchiveFiles`, `hashInCategory`, `maybeRenameArchiveTorrent`, `waitTorrentFiles`, `matchTorrentFileIndexes`, `TorrentFileForTitle`, … |
| `internal/download/archive_select_test.go` | **New** |
| `internal/download/archive_torrent_test.go` | **New** (commit `24b5719`) |
| `internal/download/util.go` | Path helpers |
| `internal/qbit/client.go` | `AddTorrentOpts` (paused/stopped), `SetFilePriority`, `RenameTorrent`, `TorrentFile.Progress` |
| `internal/qbit/client_test.go` | Tests for above |
| `internal/api/requests.go` | `selectFiles` on download-for-request |
| `cmd/gamarr/main.go` | `selectFiles` on auto-download |

#### 3. Per-ROM import from shared archive hash
| Path | Status |
|------|--------|
| `internal/download/manager.go` | `jobFileReady`, `importReadyHashJobs` (was `importArchiveHashJobs` in committed HEAD; renamed in uncommitted), `resolveImportContentPath`, `torrentFileContentPath`, `torrentWantedComplete`, `jobsOnHash`, watcher/recovery |
| `internal/download/archive_import_test.go` | **New/growing** — path strip, per-ROM import, dedup tests |
| `internal/download/nzb.go` | `jobCompleted` helper; `RetryJob` uses `jobFileReady`; NZBGet recovery for `interrupted` |
| `internal/download/nzb_test.go` | Test updates |
| `internal/download/manager_test.go` | Dedup, recovery watcher, give-up error field |

#### 4. Archive downloads API + UI
| Path | Status |
|------|--------|
| `internal/models/models.go` | `DownloadFile` struct; `DownloadEntry.Type` = `"archive"` |
| `internal/api/api.go` | `buildArchiveEntry`, `buildMergedJobEntry`, `mergedJobProgress` (uncommitted), archive grouping in `handleDownloads` |
| `internal/api/handlers_crud_test.go` | `TestDownloadsGroupsArchiveJobsByHash`, progress test (uncommitted) |
| `internal/api/openapi.json`, `router_test.go` | Minor |
| `web/static/js/app.js` | Archive cards, collapsible file list, per-file progress/rows |
| `internal/db/library.go` | `source_type` comment includes `minerva` |

#### 5. Job lifecycle (normalizeJobInvariants, recovery, dedup)
| Path | Commits / state |
|------|-----------------|
| `internal/db/db.go` | `normalizeJobInvariants`, `jobLostToRestart`, `jobClientOwned`, boot persist (`be701f0`, `2240d04`) |
| `internal/db/db_test.go` | Stale-error tests + uncommitted `TestJobStore_ClearsErrorOnOrganizingViaSingleUpdate` |
| `internal/download/manager.go` | Recovery merge, dedup (`findActiveJobByHashTitle`), error field on give-up (partial uncommitted) |
| `cmd/gamarr/main.go` | Uncommitted: drop separate `RecoverActiveTorrentJobs` goroutine (called from `RecoverOrphanedTorrents` instead) |
| `web/static/js/app.js` | Uncommitted: show `error` on completed archive rows |

**Committed fix commits (after merge `45440b3`):**

| SHA | Message | Files |
|-----|---------|-------|
| `24b5719` | fix(minerva): rename archive torrents to platform in qBittorrent | manager, qbit |
| `93e96a1` | fix(minerva): strip torrent root from archive file paths | manager, archive_import_test |
| `a6505d0` | fix(minerva): import archive ROM jobs and recover shared watchers | main, api, manager, archive_import_test |
| `5754fa8` | fix(jobs): clear stale restart error when imports complete | manager, nzb, app.js |
| `705d2c6` | fix(minerva): allow retry when archive ROM file is complete | manager, archive_import_test |
| `be701f0` | fix(jobs): auto-clear error when status leaves failure states | db, manager |
| `2240d04` | fix(jobs): persist stale-error cleanup on boot | db |

#### 6. SABnzbd in-process NZB fetch (VPN/DNS fix)
| Path | Status |
|------|--------|
| `internal/sabnzbd/client.go` | `fetchNZB`, `addNZBFile`, `AddNZBByURL` in-process upload |
| `internal/sabnzbd/client_test.go` | Tests |

#### 7. Minor / refactor (low conflict)
| Path | Notes |
|------|-------|
| `internal/scheduler/scheduler.go` | Uses `search.SortByScore` instead of inline sort — fine, keep |
| `internal/search/vimm.go`, `vimm_test.go` | **Regress vs upstream** — rebase must prefer upstream |
| `tests/e2e_test.py` | Small tweak |

### Uncommitted working tree (not yet on `dev`)

**Recommend one commit before PR split:**

```
fix(minerva): per-ROM import readiness, dedup, and download progress
```

| File | Change summary |
|------|----------------|
| `internal/download/manager.go` | `importReadyHashJobs` (per-file ready), dedup, recovery ordering, give-up sets `error` |
| `internal/download/manager_test.go` | Dedup, recovery watcher, give-up error tests |
| `internal/download/archive_import_test.go` | Path-without-root, sibling skip, importReady tests |
| `internal/api/api.go` | `mergedJobProgress`, per-file progress in archive cards |
| `internal/api/handlers_crud_test.go` | Missing-file progress = 0 test |
| `internal/db/db_test.go` | normalize via single `Update` test |
| `cmd/gamarr/main.go` | Single recovery path (no duplicate goroutine) |
| `web/static/js/app.js` | Show errors on completed archive file rows |

---

## B. Dev tree verification

| Check | Result |
|-------|--------|
| Branch | `dev`, up to date with `origin/dev` |
| Commits ahead of `upstream/main` | 8 (counted by `git log upstream/main..dev`) |
| Uncommitted | 8 files, +322/−32 |
| `go test ./...` | **PASS** (all packages ok) |
| homelab-casc pin | `compose.yaml` → `klopstack/gamarr.git#dev` |
| Minerva in tree | `internal/search/minerva.go` present; absent from upstream |
| normalizeJobInvariants | Committed in `be701f0`/`2240d04`; extra test uncommitted |

---

## C. PR split plan (execution order)

### Phase 0 — Rebase (required, not a PR)

1. `git fetch upstream && git rebase upstream/main` on `dev`
2. Resolve conflicts — priority: keep upstream `vimm.go`, merge Minerva blocks into `manager.go`
3. Run `go test ./...`
4. Commit uncommitted fixes as one commit (see above)
5. Optionally force-push `origin/dev` (homelab pin will pick it up)

**Likely conflict files:** `manager.go`, `vimm.go`, `vimm_test.go`, `nzb.go`, `db.go`, `qbit/client.go`, `app.js`, `README.md`

---

### PR 1 — Minerva search indexer

| Field | Value |
|-------|-------|
| **Title** | `feat(search): add Minerva Archive browse indexer` |
| **Branch** | `feat/minerva-search` |
| **Base** | `upstream/main` |
| **Size** | **L** (~600 LOC new + wiring) |
| **Depends on** | Phase 0 rebase only |

**Files:**
- `internal/search/minerva.go`, `minerva_test.go`
- `internal/sources/defaults.json`, `sources.go`, `load.go`, `sources_test.go`
- `internal/search/filter.go`, `filter_test.go`, `scoring.go`, `scoring_test.go`, `sourcename.go`, `health.go`, `health_test.go`, `registry_flow_test.go`
- `internal/api/admin.go`, `search_indexers_test.go`, `torznab_wire.go`
- `cmd/gamarr/main.go` (SearchMinerva goroutine hunks only)
- `internal/api/requests.go` (SearchMinerva goroutine hunks only)
- `README.md` (Minerva sections)
- `web/static/js/app.js` (isMinerva search result badge only)

**Description:** Loads per-console Minerva browse pages, caches hourly, matches locally, returns shared archive magnet hits ranked above Prowlarr.

**Test plan:**
- [ ] `go test ./internal/search/... ./internal/sources/...`
- [ ] Manual search for a known GB title; verify Minerva results appear
- [ ] Indexer health page shows Minerva OK

---

### PR 2 — qBittorrent file-priority API extensions

| Field | Value |
|-------|-------|
| **Title** | `feat(qbit): paused add, file priorities, and rename` |
| **Branch** | `feat/qbit-archive-ops` |
| **Base** | `upstream/main` |
| **Size** | **S** (~120 LOC) |
| **Depends on** | None (parallel with PR 1). Builds on merged PR #44 API key. |

**Files:**
- `internal/qbit/client.go` — `AddTorrentOpts`, `SetFilePriority`, `RenameTorrent`, `TorrentFile.Progress`
- `internal/qbit/client_test.go`

**Description:** Extends qBit client for selective multi-file torrent downloads (paused add, per-file priority, rename display name).

**Test plan:**
- [ ] `go test ./internal/qbit/...`
- [ ] Integration: add paused torrent, set priorities, resume

---

### PR 3 — Minerva selective archive download

| Field | Value |
|-------|-------|
| **Title** | `feat(minerva): selective file download from archive magnets` |
| **Branch** | `feat/minerva-archive-download` |
| **Base** | `upstream/main` |
| **Size** | **L** (~400 LOC in manager) |
| **Depends on** | **PR 2** (qBit client). Can include PR 1 wiring for `selectFiles` detection. |

**Files:**
- `internal/download/manager.go` — `selectArchiveFiles`, `DownloadTorrent` signature, `hashInCategory`, `maybeRenameArchiveTorrent`, `waitTorrentFiles`, `matchTorrentFileIndexes`, `TorrentFileForTitle`, …
- `internal/download/archive_select_test.go`, `archive_torrent_test.go`
- `internal/download/util.go`
- `cmd/gamarr/main.go`, `internal/api/requests.go` — `selectFiles` flag wiring
- Commit hunks from `24b5719`, `93e96a1`

**Description:** Minerva magnets add paused; matching ROM files get priority 1, torrent renamed to platform, then started.

**Test plan:**
- [ ] `go test ./internal/download/... -run Archive`
- [ ] Live: download one GB zip from Minerva; verify only selected file downloads

---

### PR 4 — Per-ROM import from shared hash

| Field | Value |
|-------|-------|
| **Title** | `feat(minerva): import ROMs independently from shared archive torrent` |
| **Branch** | `feat/minerva-per-rom-import` |
| **Base** | `upstream/main` |
| **Size** | **L** |
| **Depends on** | **PR 3** |

**Files:**
- `internal/download/manager.go` — `jobFileReady`, `importReadyHashJobs`, `resolveImportContentPath`, `torrentFileContentPath`, watcher/recovery
- `internal/download/archive_import_test.go`
- `internal/download/nzb.go` — `jobCompleted`, `RetryJob`/`jobFileReady`, NZBGet interrupted recovery
- `internal/download/nzb_test.go`, `manager_test.go`
- Commits: `a6505d0`, `705d2c6`, uncommitted manager/import tests

**Description:** Multiple jobs share one Minerva hash; each ROM imports when its file reaches 100%, siblings continue downloading.

**Test plan:**
- [ ] `go test ./internal/download/... -run 'Archive|Ready|Import|Recover'`
- [ ] Two ROMs same hash: first completes while second still downloading

---

### PR 5 — Archive downloads API and UI

| Field | Value |
|-------|-------|
| **Title** | `feat(api): group archive torrent jobs with per-file progress` |
| **Branch** | `feat/archive-downloads-ui` |
| **Base** | `upstream/main` |
| **Size** | **M** |
| **Depends on** | **PR 4** (progress semantics). Can open as draft in parallel. |

**Files:**
- `internal/models/models.go`
- `internal/api/api.go` — `buildArchiveEntry`, `mergedJobProgress`, `handleDownloads` grouping
- `internal/api/handlers_crud_test.go` — archive + progress tests
- `internal/api/openapi.json`, `router_test.go` (if changed)
- `web/static/js/app.js` — archive cards, collapsible file list
- `internal/db/library.go` (source_type comment)
- Commit `a6505d0` api hunks + uncommitted api/js/tests

**Description:** Downloads page groups jobs by torrent hash into archive cards with per-ROM progress, status, retry.

**Test plan:**
- [ ] `go test ./internal/api/... -run Download`
- [ ] UI: multi-ROM archive shows nested file rows; collapse state persists across poll

---

### PR 6 — Job invariant normalization and boot cleanup

| Field | Value |
|-------|-------|
| **Title** | `fix(jobs): normalize error fields and smart restart recovery` |
| **Branch** | `fix/job-invariants-recovery` |
| **Size** | **M** |
| **Depends on** | None strictly; land before or with PR 4/7 |

**Files:**
- `internal/db/db.go` — `normalizeJobInvariants`, `jobLostToRestart`, `jobClientOwned`, boot persist
- `internal/db/db_test.go`
- Commits: `be701f0`, `2240d04`

**Description:** Non-error statuses auto-clear stale `error`; client-owned downloads survive restart as `downloading`; in-process work becomes `interrupted`; normalized rows persist on boot.

**Test plan:**
- [ ] `go test ./internal/db/...`
- [ ] Restart with active qBit job → stays downloading, not interrupted

---

### PR 7 — Minerva job lifecycle polish

| Field | Value |
|-------|-------|
| **Title** | `fix(minerva): dedup jobs, merge recovery, and stale error UI` |
| **Branch** | `fix/minerva-job-lifecycle` |
| **Size** | **M** |
| **Depends on** | **PR 4, PR 5, PR 6** |

**Files:**
- All uncommitted changes (see § uncommitted)
- Commit hunks: `5754fa8`, parts of `a6505d0`/`705d2c6` not in PR 3–5

**Description:** Dedup active archive jobs by hash+title; single recovery path; per-ROM progress in API; show errors on completed archive rows; import give-up writes `error` field.

**Test plan:**
- [ ] `go test ./...`
- [ ] Re-download same Minerva title → same job ID
- [ ] Completed archive file with past error shows error text

---

### PR 8 — SABnzbd in-process NZB upload

| Field | Value |
|-------|-------|
| **Title** | `fix(sabnzbd): fetch NZB in-process for VPN/container DNS` |
| **Branch** | `fix/sabnzbd-inprocess-nzb` |
| **Size** | **S** |
| **Depends on** | None — **can merge independently, any order** |

**Files:**
- `internal/sabnzbd/client.go`, `client_test.go`

**Description:** Gamarr fetches NZB from Prowlarr and uploads bytes to SAB; avoids SAB fetching Docker-internal URLs when on VPN.

**Test plan:**
- [ ] `go test ./internal/sabnzbd/...`
- [ ] Usenet download via SAB in VPN compose setup

---

## Dependency graph

```
Phase 0 (rebase)
    │
    ├── PR 1 (minerva search) ─────────────────────────┐
    ├── PR 2 (qbit ops) ──► PR 3 (archive download) ──► PR 4 (per-ROM import) ──► PR 5 (archive UI)
    ├── PR 6 (job invariants) ─────────────────────────┤
    ├── PR 8 (sabnzbd) ─ independent ──────────────────┤
    └── upstream vimm fixes (no PR — take from main) ───┘
                              │
                              ▼
                         PR 7 (lifecycle polish) — after 4+5+6
```

**Suggested merge order:** PR 6 → PR 2 → PR 1 → PR 3 → PR 4 → PR 5 → PR 7 → PR 8 (8 can go anytime)

---

## D. Flags and risks

### Should NOT go upstream
- Nothing in the gamarr codebase is homelab-specific.
- `homelab-casc` compose build pin stays in homelab repo only.

### Merge conflict hotspots
| File | Why |
|------|-----|
| `internal/download/manager.go` | Upstream vault-archive + import-race hardening vs Minerva file-priority/import |
| `internal/search/vimm.go` | Fork behind PR #46 — **take upstream wholesale** |
| `internal/download/nzb.go` | Recovery + `jobCompleted` vs upstream organize paths |
| `internal/db/db.go` | Restart semantics may overlap upstream job load |
| `web/static/js/app.js` | Downloads UI + search badges |
| `README.md` | Both sides edited env var docs |

### Rebase recommendation
**Yes — mandatory.** Fork snapshot predates upstream PRs #42–46. Without rebase, PRs will reintroduce regressions (Vimm rate limit) and duplicate/conflict vault logic.

### qBittorrent API key
Already upstream via PR #44. PR 2 only adds archive-specific client methods; verify no duplicate `NewWithAPIKey` / auth logic when rebasing.

---

## E. Commit-on-dev checklist (before splitting)

```bash
cd /tmp/gamarr-dev
git add cmd/gamarr/main.go \
  internal/api/api.go internal/api/handlers_crud_test.go \
  internal/db/db_test.go \
  internal/download/archive_import_test.go internal/download/manager.go internal/download/manager_test.go \
  web/static/js/app.js
git commit -m "$(cat <<'EOF'
fix(minerva): per-ROM import readiness, dedup, and download progress

Import only hash jobs whose selected file is complete; dedup active
archive downloads by hash+title; merge torrent recovery paths; show
per-ROM progress and stale errors correctly in the downloads UI.
EOF
)"
go test ./...
```

Then rebase onto `upstream/main` and cherry-pick or branch per PR above.
