---
title: Agent Mode
---

# Agent Mode <Badge type="tip" text="Beta" />

Dozzle can run in agent mode which can expose Docker hosts to other Dozzle instance. All communication is done over a secured connection using TLS. This means that you can deploy Dozzle on a remote host and connect to it from your local machine.

## How to create an agent?

To create a Dozzle agent, you need to run Dozzle with the `agent` subcommand. Here is an example:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

The agent will start and listen on port `7007`. You can connect to the agent using the Dozzle UI by providing the agent's IP address and port. The agent will only show the containers that are available on the host where the agent is running.

> [!TIP]
> You don't need to expose port 7007 if using Docker network. The agent will be available to other containers on the same network.

## How to connect to an agent?

To connect to an agent, you need to provide the agent's IP address and port. Here is an example:

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent-ip:7007
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent:7007
    ports:
      - 8080:8080 # Dozzle UI port
```

:::

Note that when connecting remotely, you don't need to mount local Docker socket. The UI will only show the containers that are available on the agent.

> [!TIP]
> You can connect to multiple agents by providing multiple `DOZZLE_REMOTE_AGENT` environment variables. For example, `DOZZLE_REMOTE_AGENT=agent1:7007,agent2:7007`.

## Setting up healthcheck

You can set a healthcheck for the agent, similar to the healthcheck for the main Dozzle instance. When running in agent mode, healthcheck checks agent connection to Docker. If Docker is not reachable, the agent will be marked as unhealthy and will not be shown in the UI.

To set up healthcheck, use the `healthcheck` subcommand. Here is an example:

```yml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 5s
      retries: 5
      start_period: 5s
      start_interval: 5s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

## Changing agent's name

Similar to Dozzle instance, you can change the agent's name by providing the `DOZZLE_HOSTNAME` environment variable. Here is an example:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:agent agent --hostname my-special-name
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_HOSTNAME=my-special-name
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

This will change the agent's name to `my-special-name` and reflected on the UI when connecting to the agent.
