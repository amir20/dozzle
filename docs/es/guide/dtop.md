---
title: Presentamos dtop
sourceHash: 3137db243510
---

# ¿Qué es dtop?

`dtop` es un complemento de línea de comandos para Dozzle que muestra en tiempo real, desde el terminal, los contenedores de Docker que se ejecutan en tu sistema. Piensa en él como un `docker ps` más completo que puedes dejar abierto en un panel de tmux, y cuando necesites el historial completo de logs, la búsqueda o los gráficos, `dtop` te lleva directamente a Dozzle.

Se conecta a los hosts de Docker mediante `ssh`, `tcp` o un `unix socket` local, así que encaja bien con las mismas configuraciones multihost que admite Dozzle.

![captura de dtop](https://github.com/amir20/dtop/raw/master/demo.gif)

## Instalación

Instálalo con Homebrew:

```bash
brew install dtop
```

O ejecútalo con Docker sin instalar nada:

```bash
docker run -v /var/run/docker.sock:/var/run/docker.sock -it ghcr.io/amir20/dtop:latest
```

Tienes las instrucciones de instalación completas en [https://github.com/amir20/dtop](https://github.com/amir20/dtop?tab=readme-ov-file#installation).

## Estado del proyecto

`dtop` es un proyecto nuevo y no tiene tantas funciones como Dozzle. Aun así, sigo trabajando activamente en añadir más. Yo lo uso a diario para vigilar todos mis contenedores en varios hosts desde la línea de comandos. Si tienes sugerencias, abre una incidencia en [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
