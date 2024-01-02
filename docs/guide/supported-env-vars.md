---
title: Environment variables and configuration
---

# Environment variables and configuration

Configurations can be done with flags or environment variables. The table below outlines all supported options and their respective env vars.

| Flag                        | Env Variable                     | Default        |
| --------------------------- | -------------------------------- | -------------- |
| `--addr`                    | `DOZZLE_ADDR`                    | `:8080`        |
| `--base`                    | `DOZZLE_BASE`                    | `/`            |
| `--hostname`                | `DOZZLE_HOSTNAME`                | `""`           |
| `--level`                   | `DOZZLE_LEVEL`                   | `info`         |
| `--auth-provider`           | `DOZZLE_AUTH_PROVIDER`           | `none`         |
| `--auth-header-user`        | `DOZZLE_AUTH_HEADER_USER`        | `Remote-User`  |
| `--auth-header-email`       | `DOZZLE_AUTH_HEADER_EMAIL`       | `Remote-Email` |
| `--auth-header-name`        | `DOZZLE_AUTH_HEADER_NAME`        | `Remote-Name`  |
| `--enable-actions`          | `DOZZLE_ENABLE_ACTIONS`          | false          |
| `--wait-for-docker-seconds` | `DOZZLE_WAIT_FOR_DOCKER_SECONDS` | 0              |
| `--filter`                  | `DOZZLE_FILTER`                  | `""`           |
| `--no-analytics`            | `DOZZLE_NO_ANALYTICS`            | false          |
| `--remote-host`             | `DOZZLE_REMOTE_HOST`             |                |
