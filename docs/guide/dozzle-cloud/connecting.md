---
title: Connecting Your Instance
---

# Connecting Your Instance

Linking a self-hosted Dozzle to [Dozzle Cloud](/guide/dozzle-cloud), confirming it actually connected, and fixing it when it did not.

## Link an instance

1. Open your self-hosted Dozzle and click the **cloud** icon in the top bar.
2. Click **Link instance**. You are sent to Cloud to sign in with GitHub or Google and confirm.
3. The instance appears on the Cloud dashboard within a few seconds.

There is no password to create, and no agent to install on the host.

## You do not need a public IP, an open port, or a domain

This is the most common worry, and the answer is no on all three.

Your Dozzle instance opens an **outbound** connection to Cloud and holds it open. Cloud never connects back to you, never scans for your host, and never needs to reach your address. That means it works normally when Dozzle is:

- behind NAT on a home network, with no port forwarding
- on a private RFC1918 address such as `192.168.1.50`
- on a Tailscale, WireGuard, or ZeroTier network
- behind CGNAT, where you could not port-forward even if you wanted to
- on a laptop that changes networks

No reverse proxy, dynamic DNS name, or static IP is required.

## Firewall rules

Only **outbound** access is needed. Allow your Dozzle host to reach:

```
agent.doligence.dozzle.dev:443    (TCP, outbound)
```

That single destination on port 443 is enough. If your firewall filters by hostname rather than IP, allow the hostname — the addresses behind it can change. Most home and small-office firewalls allow all outbound traffic already, so usually there is nothing to configure.

## Where rules and channels live

This trips up almost everyone, so it is worth stating plainly.

| What you want to change                                       | Where you do it                                           |
| ------------------------------------------------------------- | --------------------------------------------------------- |
| **What triggers an alert** — containers, patterns, thresholds | Self-hosted Dozzle → [Alerts](/guide/alerts-and-webhooks) |
| **Where alerts are delivered** — email, Telegram, Slack, ...  | Dozzle Cloud → [Channels](/guide/dozzle-cloud/channels)   |
| Reviewing past alerts, muting, upgrading                      | Dozzle Cloud                                              |

The rule is defined on your own instance because that is where your logs are. Delivery is configured in Cloud because that is what holds the connection to your phone. If you are looking for somewhere in Cloud to say "tell me when this container errors" and cannot find it, that is why — open your self-hosted Dozzle instead.

## Connecting more than one instance

Each instance links separately, using the same steps and the same Cloud account. Once linked, all of them appear together on the dashboard, and questions asked in chat cover every connected instance at once.

That combined view lives in Cloud, not inside any one self-hosted Dozzle. A self-hosted Dozzle shows the hosts you configured on it directly; it does not display other linked instances.

Linking several Dozzle instances is a different thing from Dozzle's own [Agent](/guide/agent) and [Remote Hosts](/guide/remote-hosts) features, which connect extra Docker hosts to a single Dozzle. Both are supported and can be combined.

The free plan links one instance at a time. See [Plans & Limits](/guide/dozzle-cloud/plans).

## Nothing is showing up

Work through these in order.

**1. Is the Dozzle container running?**
If Dozzle itself is stopped or restarting, nothing reaches Cloud.

**2. Was the link ever completed?**
Starting the link and not approving it leaves nothing behind. Redo the steps above and confirm the instance appears on the Instances page.

**3. Was the API key deleted?**
Deleting an instance's API key unlinks it permanently. There is no way to reattach the old key — link again to get a new one.

**4. Is outbound traffic being blocked?**
Restrictive networks (corporate, university, some VPS providers) may block outbound 443 to destinations not on an allowlist. See _Firewall rules_ above.

**5. Did you hit the free instance limit?**
The free plan links one instance at a time. Attempting to link a second shows a limit message instead of connecting.

**6. Is the container excluded from forwarding?**
A container labelled `dev.dozzle.cloud.min_level=disabled` sends nothing by design. If one specific container is missing while others work, check its labels. See [Your Data](/guide/dozzle-cloud/your-data).

## Letting the agent control containers

Reading logs and container state works as soon as an instance is linked. Start, stop, and restart are refused by your instance unless you enable them yourself:

::: code-group

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

```sh
docker run ... amir20/dozzle --enable-actions
```

:::

This is a setting on **your** Dozzle, not in Cloud, because it governs what your Dozzle is willing to do to your containers. Restart Dozzle after changing it. See [Actions](/guide/actions).

Once linked, see [In Your Dozzle](/guide/dozzle-cloud/in-dozzle) for what appears in your own UI.

## Unlinking

Delete the instance's API key on the Instances page in Cloud. The connection drops, no further data is forwarded, and your self-hosted Dozzle keeps working exactly as before. Linking never changes local log viewing.
