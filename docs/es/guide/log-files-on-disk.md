---
title: Seguir archivos de log en disco
sourceHash: e6e23f2438f9
---

# Seguir archivos de log en disco

Algunos contenedores escriben sus logs en archivos en lugar de en `stdout` o `stderr`. Dozzle solo puede leer lo que captura el propio Docker, es decir `stdout` y `stderr`, igual que `docker logs`. Los archivos que hay dentro de un contenedor no son visibles para otros contenedores, así que Dozzle no tiene forma de llegar a ellos.

## Escribe en los flujos estándar

La mejor solución es dejar de escribir en archivos. Casi todas las aplicaciones tienen una opción de configuración para enviar los logs a la consola, y la [twelve factor app](https://12factor.net/logs) explica por qué ese es el valor por defecto correcto.

Si no puedes configurar la aplicación, crea un enlace simbólico del archivo de log a la salida estándar del contenedor en tu `Dockerfile`. Es lo que hace la imagen oficial de nginx:

```dockerfile
RUN ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log
```

## Seguir un archivo con un sidecar

Cuando ninguna de las dos opciones es viable, levanta un contenedor pequeño de Alpine que siga el archivo y deje que Docker capture la salida. Dozzle lo muestra entonces como cualquier otro contenedor.

::: code-group

```sh [docker run]
docker run -d \
  --name system-log \
  --label dev.dozzle.name=system-log \
  --network none \
  --restart unless-stopped \
  --log-opt max-size=10m --log-opt max-file=3 \
  -v /var/log:/logs:ro \
  alpine tail -n 1000 -F /logs/system.log
```

```yaml [docker-compose.yml]
services:
  system-log:
    container_name: system-log
    image: alpine
    volumes:
      - /var/log:/logs:ro
    command:
      - tail
      - -n
      - "1000"
      - -F
      - /logs/system.log
    labels:
      dev.dozzle.name: system-log
    logging:
      options:
        max-size: 10m
        max-file: "3"
    network_mode: none
    restart: unless-stopped
```

:::

La versión con Compose es útil si quieres que el flujo de logs sobreviva a un reinicio del servidor. En las pruebas, Alpine consumió unos `~50KB` de memoria.

### Por qué `-F` y no `-f`

`tail -f` sigue el descriptor de archivo abierto. Cuando el archivo rota, ese descriptor apunta al archivo antiguo ya renombrado y el flujo se queda mudo. `tail -F` sigue la ruta y vuelve a abrir el archivo tras una rotación, así que no deja de funcionar.

Por el mismo motivo, monta el **directorio** y no el archivo. Un bind mount de un único archivo queda ligado al inodo de ese archivo, así que una rotación en el host sustituye el archivo y el contenedor sigue mirando al antiguo, incluso con `-F`.

### Cargar el historial

Docker solo guarda lo que el contenedor ha impreso desde que arrancó, así que reiniciar el sidecar borra todo lo que Dozzle tenía. `-n 1000` imprime las últimas 1000 líneas al arrancar para que la vista no salga vacía.

### Varios archivos

Cuando se le pasa más de un archivo, `tail` antepone el nombre del archivo a cada bloque. Los comodines necesitan una shell, ya que la imagen no tiene entrypoint que los expanda:

```sh
docker run -d -v /var/log:/logs:ro alpine sh -c 'tail -n 1000 -F /logs/*.log'
```

La etiqueta `dev.dozzle.name` de arriba le da al sidecar un nombre legible en la interfaz. Consulta [Nombres de contenedores](/es/guide/container-names) para más detalles.
