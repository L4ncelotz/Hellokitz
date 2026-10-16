# Roadmap — HelloKit

**Status:** Outcome-based proposal, without fixed delivery dates. Tasks are NOT already completed.

## Phase 1 — HelloKit CLI MVP (Windows)

**Goal:** Recreate a selected, reviewable environment on a fresh eligible Windows machine.

- Go native binary, help/version/doctor.
- Profile schema and storage, Estzdev editable sample and Custom flow.
- WinGet apps; optional version-managed toolchains through mise.
- Explicit approved config changes / editor extensions only for supported adapters.
- Preview, preflight, deterministic execution, cancellation, verification and reports.
- Safe second run; explicit unknown/blocked states.

**Release test:** clean VM -> profile preview -> apply -> verify -> rerun -> no unexpected changes.

## Phase 2 — Stronger local profile workflow

**Goal:** Reduce manual profile upkeep while preserving control.

- Guided capture of supported installed tools (review required before save).
- Compare a machine with a profile (drift report, read-only).
- Profile revisions and local import/export compatibility policy.
- Better coverage of developer tools and editors, subject to stable adapters.
- Project bootstrap/templates as a **separate bounded command**, not mixed with machine setup.

**Release test:** a capture does not export secrets; diff explains precisely why two machines differ; project setup cannot silently change global machine state.

## Phase 3 — HelloKit Cloud (optional)

**Goal:** Edit and select profiles from a website without making the CLI depend on that service.

- Web dashboard for versioned profile CRUD.
- Standards-based browser-assisted CLI login and session/token storage.
- Explicit remote profile selection and revision pinning / conflict rules.
- Download -> validate -> preview -> confirm -> apply pipeline identical to local.
- Clear separation between public templates and private user profiles.

**Release test:** offline CLI still works; malicious/invalid remote profile rejected; remote edits cannot trigger automatic installation on existing machines.

## Phase 4 — Broader ecosystem

**Goal:** Expand coverage only after proven demand.

- Linux/macOS adapters; separate OS-specific profile targeting.
- Community profile catalog with moderation and provenance.
- Team/shared profiles if access controls and audit requirements justify them.
- Optional richer visualization or desktop UI if CLI usability data shows need.

## Explicit non-goals unless reconsidered

- Writing a replacement for WinGet/mise or a general-purpose OS configuration management platform.
- Copying everything from an existing machine including accounts, browser profiles or secrets.
- Running remote scripts from profiles or allowing websites to execute commands on user machines.
- Unattended privilege escalation, forced restart, total rollback guarantee.

## Prioritization guidelines

Prioritize **correctness + user trust** above feature count. New features should demonstrate one of: better reproducibility, fewer user actions, safer behavior or reduced maintenance. Do not add a microservice, external database or new runtime merely because it is interesting.

Detailed technical tasks are tracked in [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md).
