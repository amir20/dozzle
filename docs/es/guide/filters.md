---
title: Filtros
sourceHash: e380a612fd7f
---

# Filtrar contenedores

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle admite filtrado condicional similar al [--filter](https://docs.docker.com/reference/cli/docker/container/ls/#filter) de Docker mediante `DOZZLE_FILTER` o `--filter`. Los filtros se pasan directamente a Docker para limitar lo que Dozzle puede ver. Por ejemplo, se puede filtrar por etiqueta con `--filter "label=color"`, igual que harías con `docker ps --filter "label=color"`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --filter label=color
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_FILTER: label=color
```

:::

Los filtros más habituales son `name` o `label`, para limitar a qué contenedores accede Dozzle.

## Filtros de interfaz, de agente y de usuario

Dozzle admite varios filtros para limitar los contenedores que puede ver. Se pueden definir a nivel de interfaz, de agente o de usuario.

1. **Filtros de interfaz**: se aplican a la instancia de la interfaz de Dozzle y se envían a Docker para restringir los contenedores visibles. Afectan a todos los agentes y a los usuarios que no tengan sus propios filtros.
2. **Filtros de agente**: se definen a nivel de agente y se envían a Docker para limitar los contenedores que ese agente expone. Los filtros de agente y los de interfaz se combinan para restringir los contenedores.
3. **Filtros de usuario**: se definen a nivel de usuario y determinan qué contenedores puede ver ese usuario. Si no hay filtros de usuario, Dozzle usa por defecto los filtros de interfaz.

Para más información sobre cómo definir filtros para usuarios concretos, consulta [filtros de usuario](/es/guide/authentication#setting-specific-filters-for-users). Para los detalles sobre los filtros de agente, consulta [filtros de agente](/es/guide/agent#setting-up-filters).

> [!WARNING]
> Es importante entender que los distintos filtros se combinan para limitar los contenedores. Por ejemplo, si defines `--filter label=color` a nivel de interfaz y `--filter label=type` a nivel de agente, Dozzle solo mostrará los contenedores que tengan las dos etiquetas, `color` y `type`.
