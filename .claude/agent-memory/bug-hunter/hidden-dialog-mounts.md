---
name: hidden-dialog-mounts
description: Components inside a closed <dialog> gated only on shared data (not on open state) still mount and run their setup side effects
metadata:
  type: project
---

Dialogs in this app (AddHostModal, StepModal) are always in the DOM; a child gated by `v-if="status"` mounts as soon as the shared `useSetup().status` is loaded (wizard auto-open on fresh installs, visiting /settings), even with the dialog closed.

**Why:** Found 2026-09-27: AddHostPanel's immediate watch POSTs /api/setup/agent-cert on mount, so the hidden modal in HostMenu created the private agent key pair on every fresh install and kept the key in memory all session.

**How to apply:** When reviewing a dialog child with fetches/POSTs in setup or `immediate` watchers, check whether its v-if includes the dialog's open state. Related: [[setup-shared-status]] (module-level status in composable/setup/setup.ts shared by wizard, settings, AddHostModal).
