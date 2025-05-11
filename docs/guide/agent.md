---
title: Agent Mode
---

# Agent Mode <Badge type="warning" text="Docker Only" />

Dozzle can run in agent mode which can expose Docker hosts to other Dozzle instances. All communication is done over a secured connection using TLS. This means that you can deploy Dozzle on a remote host and connect to it from your local machine.

> [!NOTE] Using Docker Swarm?
> If you are using Docker Swarm Mode, you don't need to use agents. Dozzle will automatically discover itself and create a cluster using swarm mode. See [Swarm Mode](/guide/swarm-mode) for more information.

## How to Create an Agent

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

> [!NOTE] Docker Socket Proxy users
> If you are using a remote agent you **CANNOT** add a socket proxy on top of the agent. Dozzle agents **REPLACE** using a proxy, see [Remote Hosts](/guide/remote-hosts.md) for more info and how to use a socket proxy instead of an agent.


The agent will start and listen on port `7007`. You can connect to the agent using the Dozzle UI by providing the agent's IP address and port. The agent will only show the containers that are available on the host where the agent is running.

> [!TIP]
> You don't need to expose port 7007 if using Docker network. The agent will be available to other containers on the same network.

## How to Connect to an Agent

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

## Common Issues

### Agent Not Showing Up

If you are seeing `An agent with an existing ID was found. Removing the duplicate host.` then you have two hosts that use the same Server ID.

Dozzle utilizes the Docker API to collect information about hosts. Each agent requires a unique host ID that remains consistent across restarts to ensure proper identification. Currently, agents identify the host using either Docker's system ID or node ID.

If you are operating in a Swarm environment, the node ID will be employed for this purpose. However, if you notice that not all hosts are visible, it may be due to the presence of duplicate hosts configured with the same host ID.

To resolve this issue, you should remove `/var/lib/docker/engine-id` from your system and restart. This action will help eliminate any conflicts caused by duplicate host IDs. For additional information and troubleshooting tips, please refer to the [FAQ](/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it).

## Advanced Options

### Setting Up Healthcheck

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

### Changing Agent's Name

Similar to Dozzle instance, you can change the agent's name by providing the `DOZZLE_HOSTNAME` environment variable. Here is an example:

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent --hostname my-special-name
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

This will change the agent's name to `my-special-name` and will be reflected on the UI when connecting to the agent.

### Setting Up Filters

You can set up filters for the agent to limit the containers it can access. These filters are passed directly to Docker, restricting what Dozzle can view.

```yaml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_FILTER=label=color
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

This will restrict the agent to displaying only containers with the label `color`. Keep in mind that these filters are combined with the UI filters to narrow down the containers. To learn more about the different types of filters, read the [filters documentation](/guide/filters#ui-agents-and-user-filters).

### Custom Certificates

By default, Dozzle uses self-signed certificates for communication between agents. This is a private certificate which is only valid to other Dozzle instances. This is secure and recommended for most use cases. However, if Dozzle is exposed externally and an attacker knows exactly which port the agent is running on, then they can set up their own Dozzle instance and connect to the agent. To prevent this, you can provide your own certificates.

To provide custom certificates, you need to mount or use secrets to provide the certificates. Here is an example:

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: cert
        target: /dozzle_cert.pem
      - source: key
        target: /dozzle_key.pem
    ports:
      - 7007:7007
secrets:
  cert:
    file: ./cert.pem
  key:
    file: ./key.pem
```

> [!TIP]
> Docker secrets are preferred for providing certificates. They can be created using `docker secret create` command or as the example above using `docker-compose.yml`. The same certificates should be provided to the Dozzle instance connecting to the agent.

This will mount the `cert.pem` and `key.pem` files to the agent. The agent will use these certificates for communication. The same certificates should be provided to the Dozzle instance connecting to the agent.

To generate certificates, you can use the following command:

```sh
$ openssl genpkey -algorithm Ed25519 -out key.pem
$ openssl req -new -key key.pem -out request.csr -subj "/C=US/ST=California/L=San Francisco/O=My Company"
$ openssl x509 -req -in request.csr -signkey key.pem -out cert.pem -days 365
```

## Comparing Agents with Remote Connection

Agents are similar to remote connections, but they have some advantages. Generally, agents are preferred over remote connections due to performance and security reasons. Here is a comparison:

| Feature     | Agent                        | Remote Connection               |
| ----------- | ---------------------------- | ------------------------------- |
| Performance | Better with distributed load | Worse on the UI                 |
| Security    | Private SSL                  | Insecure or Docker TLS          |
| Ease of use | Easy out of the box          | Requires exposing Docker socket |
| Permissions | Full access to Docker        | Can be controlled with a proxy  |
| Reconnect   | Automatically reconnects     | Requires UI restart             |
| Healthcheck | Built-in healthcheck         | No healthcheck                  |
| Filters     | Supports filters             | No support for filters          |

If you do plan to use remote connections, make sure to secure the connection using Docker TLS or a reverse proxy.
