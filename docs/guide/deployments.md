---
title: Deployments
---

# Deployments

<Badge type="tip" text="Dozzle Cloud" />

Dozzle can deploy and manage Docker Compose projects directly on any host connected to [Dozzle Cloud](/guide/dozzle-cloud). Each project is stored as a git-backed repository on disk, which means every change is versioned and any previous revision can be redeployed with a single command.

> [!IMPORTANT]
> Deployments are only available through Dozzle Cloud. There is no UI in Dozzle for managing them yet — you interact with the feature by chatting with the Cloud assistant (for example: _"Deploy this compose file as `nginx-demo`"_ or _"Roll back `api` to the previous version"_).

## How It Works

When you deploy a project, Dozzle:

1. Creates a directory at `./data/stacks/{project}` on the target host
2. Writes your `compose.yaml` into it and commits it to a local git repository
3. Calls Docker Compose to bring the project up
4. On subsequent deploys, commits the new compose file and redeploys
5. Stores every revision as a git commit, so you can list history and roll back

Project names must match `[a-z0-9][a-z0-9_-]*` (lowercase letters, digits, dashes, and underscores).

## Enabling Deployments

Deployments require the `/data` directory to be persisted and the `--enable-actions` flag (same flag used for [Container Actions](/guide/actions)).

::: code-group

```sh
docker run \
  --volume=/var/run/docker.sock:/var/run/docker.sock \
  --volume=/path/to/data:/data \
  -p 8080:8080 \
  amir20/dozzle --enable-actions
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

:::

> [!NOTE]
> Deployments are only available on hosts with a local Docker daemon. They are not supported in Kubernetes mode.

## What You Can Do

Through Dozzle Cloud, you can ask the assistant to:

- **Deploy** a new project from a Compose YAML, or update an existing one
- **List versions** for a project and see its full commit history (hash, message, timestamp)
- **Roll back** a project to any previous commit using its short or full hash
- **Remove** a project — tears down containers and networks, and optionally deletes named volumes

Because every deploy is committed, updating a project is safe: if the new configuration breaks, you can roll back to the previous working revision.

## On-Disk Layout

Projects live under `./data/stacks/` by default:

```
data/stacks/
├── api/
│   ├── .git/
│   └── compose.yaml
└── nginx-demo/
    ├── .git/
    └── compose.yaml
```

Each directory is a standalone git repository. You can inspect history with standard git tooling if needed, but day-to-day management is meant to go through Dozzle Cloud.

> [!WARNING]
> Removing a project deletes its directory, including its git history. If you pass the "remove volumes" option, named volumes declared by the project are deleted as well — this is destructive and not recoverable.
