---
title: Acceso a la shell del contenedor
sourceHash: 267b2a6665a0
---

# Conectarse y ejecutar comandos de shell

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle permite conectarse a un contenedor o ejecutar comandos dentro de él. Ofrece una interfaz web para interactuar con los contenedores de Docker, de modo que puedes engancharte a contenedores en ejecución y ejecutar comandos directamente desde el navegador. Es especialmente útil para depurar y resolver problemas en aplicaciones en contenedores. Esta función está **desactivada** por defecto porque puede suponer un riesgo de seguridad. Para activarla, define la variable de entorno `DOZZLE_ENABLE_SHELL` como `true`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --enable-shell
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
      DOZZLE_ENABLE_SHELL: true
```

:::

> [!NOTE]
> El acceso a la shell debería funcionar con todos los tipos de contenedor, incluidos Docker, Kubernetes y otras plataformas de orquestación.

## <Icon icon="mdi:shield-lock-outline" inline /> Seguridad

Cualquiera que pueda llegar a la interfaz de Dozzle podrá abrir una shell dentro de tus contenedores, lo que equivale a un `docker exec`. Antes de activar `--enable-shell` en un Dozzle accesible públicamente, ponlo detrás de [autenticación](/es/guide/authentication). Los permisos por rol permiten restringir el acceso a la shell a determinados usuarios.

## <Icon icon="mdi:kubernetes" inline /> Kubernetes

En modo k8s, el acceso a la shell usa la API de Kubernetes en lugar de `docker exec`. El pod de destino debe incluir una shell ejecutable (`/bin/sh`, `/bin/bash`, etc.). Las imágenes mínimas construidas `FROM scratch` o las imágenes distroless sin shell no admiten la conexión.
