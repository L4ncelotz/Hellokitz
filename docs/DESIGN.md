# Product & Experience Design — HelloKit

**Status:** Design direction, not shipped UI. Primary target is a **terminal experience**; the web design below is future scope.

## 1. Identity

- **Name:** HelloKit
- **Promise:** "One command. Your dev environment."
- **Voice:** Welcoming at entry; precise and serious while touching a user's machine.
- **Developer reference:** The first `Hello, World!` is a familiar start; HelloKit makes that first line easier on each new machine.
- **Primary design metric:** A developer should know **what will change and why** before pressing Confirm.

## 2. Interaction principles

1. **Progressive disclosure:** Start with one choice (Estzdev / Custom / Local File); add details only when relevant.
2. **Review before mutate:** Plan screen is the default approval surface.
3. **Truthful status:** `Already installed`, `Cannot verify`, `Blocked`, `Installing`, `Verified`, and `Failed` are distinct.
4. **Resilient re-entry:** Re-running is normal; no shame/error for packages already installed.
5. **Accessible by design:** Keyboard-first, no color-only distinctions, reduced motion, screen-reader prompts and non-TTY text output.
6. **No fake completion:** Checkmarks require real validation; post-install verification matters.

## 3. Terminal screen map

```text
START
  |
  +-- [Estzdev] ----------+
  |                       |
  +-- [Custom Wizard] ----+--> PROFILE SUMMARY
  |                       |          |
  +-- [Local YAML] -------+          v
                                   PREFLIGHT
                                      |
                                      v
                                PLAN / DIFF
                                      |
                                  CONFIRM
                                 /       \
                             CANCEL     APPLY
                                          |
                                      VERIFY
                                          |
                                        REPORT
```

Every meaningful step should support Cancel/Back before apply. Confirmation is explicitly **off by default**.

### Setup summary (mock)

```text
Hello, developer!

Profile: Estzdev   [Local sample]
Host: Windows x64  |  WinGet: available

  2 apps requested
  1 already satisfied
  1 installation required
  0 blocked

[View exact actions]  [Continue]  [Cancel]
```

Summary displays **both** total and actionable counts, and provides detail on package source and version before applying.

### Outcome summary (mock)

```text
HelloKit setup finished: PARTIAL

  Verified  1
  Skipped   1
  Failed    1

VS Code: Download failed (connection timeout)
Next: reconnect and rerun `hellokit setup --profile estzdev`
```

No `Success!` headline when any required step failed. A sample completion state is never presented as a real tool run.

## 4. TUI style tokens (provisional)

- **Typography:** Native terminal monospace. Avoid ASCII art banners that disrupt screen readers.
- **Visual hierarchy:** One top-level prompt/question at a time; compact instructions and explicit focus state.
- **Color:** Neutral base; green success, amber warning, red error only as *additional* cues. Offer `--no-color` and respect `NO_COLOR` where relevant.
- **Motion:** Optional short progress indicators. Detailed execution stays in readable append-only output.
- **Density:** Summary first, expandable per-item details. Avoid forcing users through long lists for a small setup.
- **Copy:** `Will install` / `Already satisfied` / `Cannot check version` / `Requires approval` / `Not changed`.

## 5. Future web dashboard (Phase 3, not MVP)

Information architecture proposal:

```text
Dashboard
  Profiles
    My Profiles
    Profile Detail
      Apps / Runtimes / Settings
      Revision History
      Export Profile
    New Profile
  Account / Sessions
```

### Profile editor UX

- Side navigation: Apps, Runtimes, Editor, Supported Settings.
- Main area: sortable selection table with tool name, manager, target version and availability.
- Right summary: plan-like diff **of profile edits**, not a claim that any machine has changed.
- Save as new revision, preview YAML and show schema validation errors inline.
- Sensitive information fields are **not supported**; no password/token editor.

### Cross-device UX

- Website shows `Profile saved — revision 7` rather than `Your PC is ready`.
- CLI independently downloads that revision, verifies data, previews and executes locally.
- A modified remote profile cannot initiate local execution by itself.

## 6. Accessibility and localization

- English terminal messages initially; message catalog/i18n support can be considered after wording stabilizes.
- Do not localize machine-readable JSON/report keys or profile schema property names.
- Contrast / focus visibility must be checked for dark and light terminal backgrounds.
- `NO_COLOR`, redirected output and accessible forms should receive verification cases.

## 7. Design acceptance

- User understands changes without reading source YAML.
- Unknown inventory is not visually confused with installed.
- Cancel before apply is side-effect-free.
- Full page/screen redrawing is avoidable for accessibility.
- Estzdev sample content is clearly identified as editable example, not presumed personal state.
