---
title: dtop CLI
sourceHash: 88753cae7439
---

# dtop

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

## Alcance

`dtop` es deliberadamente más pequeño que Dozzle. Cubre desde el terminal la pregunta de "qué se está ejecutando ahora mismo y si algo está ardiendo", y deja en manos de Dozzle todo lo que necesita un navegador: historial de logs, búsqueda, consultas SQL y gráficos de estadísticas.

Se desarrolla en su propio repositorio y se publica con su propio calendario. Las sugerencias y los informes de errores van en [https://github.com/amir20/dtop/issues](https://github.com/amir20/dtop/issues).
