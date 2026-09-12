# Shell Script Verification Guide

> **Purpose**: avoid *false failures* and *false conclusions* in setup / verification shell scripts.
> Source of truth for these patterns: the real incidents in `09-12-setup-dev-environment` (2026-09-12).

## When to Read

- Writing or editing any shell script in this repo (`install-docker.sh`, task-level `setup-*.sh`)
- Writing gate / assertion logic that checks a tool version or an installed capability
- Debugging a script that reports failure while the command it checks clearly printed correct output

---

## Trap 1: `set -o pipefail` + an early-exiting consumer → SIGPIPE (exit 141)

**Symptom**: the gate reports failure even though the command printed exactly the expected output.

**Cause**: `head -1` / `grep -q` exit as soon as they are satisfied. The producer
(`make`, `docker --version`, …) then receives SIGPIPE and dies with status `141`.
With `pipefail`, the whole pipeline inherits `141`, so `cmd | head -1 || die` fires.

**Do not write**:

```bash
make --version | head -1 || die "make missing"          # ✗ dies with 141 despite success
docker --version | grep -q '29\.7\.2' || die "wrong"    # ✗ same hazard
apt-mark showhold | grep -qx 'docker-ce' || die "not held"
```

**Write instead** — capture first, then match without a pipe:

```bash
contains() { case "$1" in *"$2"*) return 0 ;; *) return 1 ;; esac; }

make_ver=$(make --version 2>&1) || die "make missing"
printf '%s\n' "${make_ver%%$'\n'*}"          # print first line without a pipe

docker_ver=$(docker --version 2>&1)
contains "$docker_ver" "Docker version 29.7.2" || die "docker version != 29.7.2 -> $docker_ver"

holds=$(apt-mark showhold 2>&1)
contains "$holds" "docker-ce" || die "docker-ce is not held"
```

**Real evidence**: in `09-12-setup-dev-environment`, `make` was installed correctly —
the script even printed `GNU Make 4.3` — and the very next line reported
`[FAIL] make 不可用`. Cost: one full re-run of the install script.

> Rule of thumb: **inside a `pipefail` script, any `cmd | head` or `cmd | grep -q` is a
> latent false failure.** `sed -n '1p'` is safe (it drains stdin); `head`/`grep -q` are not.

---

## Trap 2: version assertions must not assume a leading `v`

The `Makefile` pins `GOLANGCI_LINT_VERSION ?= v2.12.2` (with `v`), but the installed
binary self-reports `2.12.2` (no `v`). Asserting on the pinned string fails.

```bash
# ✗ contains "$gl_ver" "v2.12.2"   ->  false negative
GL_VER_NUM="${GOLANGCI_VERSION#v}"                 # v2.12.2 -> 2.12.2
contains "$gl_ver" "$GL_VER_NUM" || die "golangci-lint version != ${GL_VER_NUM} -> $gl_ver"
```

**Real evidence**: the same task failed its G4 gate with
`golangci-lint 版本 ≠ v2.12.2 -> golangci-lint has version 2.12.2 built with go1.26.6 …`
— the tool was correct; the assertion was wrong.

> Normalize before comparing: strip the `v`, compare numeric strings. When a tool's
> self-reported version and the repo's pinned constant can disagree in form, assert on the
> normalized form and print the raw string in the failure message.

---

## Trap 3: verify a pinned version is actually fetchable *before* writing the install script

`https://go.dev/dl/?mode=json` returns **only the newest two releases** — a pinned older
version looks "missing" when it is in fact available. Use `&include=all` to check:

```bash
curl -fsSL 'https://go.dev/dl/?mode=json&include=all' \
  | python3 -c 'import json,sys; d=json.load(sys.stdin); print([r["version"] for r in d][:5])'
```

For apt-distributed packages, check the real candidate list instead of guessing:

```bash
curl -fsSL 'https://download.docker.com/linux/ubuntu/dists/jammy/stable/binary-amd64/Packages' \
  | grep '^Version:' | sort -V | tail -5
```

**Real evidence**: the same task initially concluded "Go 1.26.6 not found" from the
two-entry `?mode=json` output. Re-checking with `include=all` (365 entries) confirmed
`go1.26.6` existed with a sha256. **A wrong negative conclusion is as costly as a false
gate failure.**

---

## Trap 4: scripts must be safely re-runnable

Re-running an install/verification script is the normal recovery path — design for it:

- `ln -sfn` (not `ln -s`) so a repeated run does not nest links
- skip a download when the artifact already matches its expected checksum
- skip an install when the binary exists *and* reports the expected version
- never `rm -rf` a path you have not first confirmed is the intended target

```bash
if [ -f "$TARBALL" ] && echo "$EXPECT_SHA  $TARBALL" | sha256sum -c - >/dev/null 2>&1; then
  ok "tarball present and checksum matches, skipping download"
else
  curl -fsSL -o "$TARBALL" "$URL" || die "download failed"
  echo "$EXPECT_SHA  $TARBALL" | sha256sum -c - || die "checksum mismatch"
fi
```

**Real evidence**: the script was re-run three times while fixing Trap 1/2; the later runs
short-circuited Docker / Go / make / golangci-lint and finished in seconds, which is what
made iterating on the gate logic cheap.

---

## Pre-Commit Checklist for a Script

- [ ] No `cmd | head` or `cmd | grep -q` in a `pipefail` script
- [ ] Every version assertion compares a **normalized** string, and the failure message prints the raw output
- [ ] Pinned versions were proven fetchable (registry/apt/`include=all`) before the script was written
- [ ] Re-running the script is non-destructive and short-circuits already-done work
- [ ] Every `die` message names **what is missing** and **how to fix it**
- [ ] Failures stop the run instead of silently continuing into later, more destructive steps
