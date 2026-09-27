---
name: setup-endpoints-csrf
description: Setup wizard POST routes decode JSON without a Content-Type check, so under auth=none they are form-CSRFable during the setup window
metadata:
  type: project
---

`decodeSetupBody` (internal/web/setup.go) JSON-decodes any body regardless of Content-Type. A cross-site `<form enctype="text/plain">` can forge a JSON body, and with auth provider NONE there is no cookie needed, so every `/api/setup/*` POST (account, auth, agents, agent-cert, restart) is CSRFable while the setup window is open. Simple auth is safe because the session cookie is SameSite=Lax.

`reportUsage` (internal/web/beacon.go) already does the right check (`mime.ParseMediaType == application/json`), so the fix is to move that check into decodeSetupBody.

**Why:** first flagged in the review of v11.1.2..HEAD (2026-09), when POST /setup/agents made it possible to persist an attacker's agent into dozzle.yml.
**How to apply:** when a new setup or state-changing POST route appears, check whether it goes through decodeSetupBody and whether the Content-Type check exists yet.
