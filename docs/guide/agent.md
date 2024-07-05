---
title: Agent Mode
---

# Agent Mode

Dozzle supports Agent Mode starting from version 8. Dozzle UI can connect to other agents with minimum setup. All communication is done over a secured connection using TLS.

## How to create an agent?

To create a Dozzle agent, you need to run Dozzle with the `agent` subcommand. Here is an example:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:agent agent
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

The agent will start and listen on port `7007`. You can connect to the agent using the Dozzle UI by providing the agent's IP address and port.

## How to connect to an agent?

To connect to an agent, you need to provide the agent's IP address and port. Here is an example:

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent-ip:7007
```

```yaml [docker-compose.yml]
// TODO hostname, healthcheck
```
