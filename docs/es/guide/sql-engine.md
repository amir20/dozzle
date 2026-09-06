---
title: Motor SQL
sourceHash: 0387115a7372
---

# Motor SQL

El motor SQL es una herramienta potente que te permite ejecutar consultas SQL sobre tus datos. Está pensado para que quien ya conoce SQL pueda trabajar con sus datos en un lenguaje familiar.

Esta función está en fase beta y está disponible para todos los usuarios. Si tienes comentarios o sugerencias, cuéntanoslo.

## Primeros pasos

Para empezar a usar el motor SQL necesitas un conjunto de datos que consultar. Solo se pueden consultar logs en JSON. Dozzle aprovecha WebAssembly para ejecutar las consultas SQL en el navegador, así que tus datos nunca salen de tu máquina.

Para empezar, asegúrate de tener logs en JSON, abre el desplegable y elige `SQL Analytics`. También hay un atajo de teclado, `Ctrl+Shift+F` (o `Cmd+Shift+F` en macOS), para abrir el motor SQL rápidamente.

## ¿Cómo funciona?

El motor SQL usa WebAssembly para ejecutar consultas SQL en el navegador con DuckDB. La primera vez que se abre, DuckDB WASM se descarga e inicializa en el navegador. Esto puede tardar si tu conexión es lenta. Después el motor SQL lee _solo_ los logs en JSON y crea una tabla virtual en DuckDB. Así puedes consultar tus datos en tiempo real.

La consulta que Dozzle ejecuta al principio es parecida a esta:

```sql
CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'
```

Esta consulta crea una tabla llamada `logs` y expande los logs en JSON en filas. A partir de ahí puedes lanzar consultas SQL sobre esa tabla para analizar tus datos.

## Consultas de ejemplo

Estas son algunas consultas que puedes ejecutar con el motor SQL:

### Contar el número de logs

```sql
SELECT COUNT(*) FROM logs
```

### Filtrar logs por un campo concreto

```sql
SELECT * FROM logs WHERE level = 'error'
```

### Agrupar logs por un campo concreto

```sql
SELECT level, COUNT(*) FROM logs GROUP BY level
```

### Consultar campos JSON anidados

```sql
SELECT message.path, message.status, message.duration
FROM logs
WHERE message.status >= 400
ORDER BY message.duration DESC
```

### Agregar por ventana temporal

```sql
SELECT
  date_trunc('minute', timestamp) AS minute,
  COUNT(*) AS error_count
FROM logs
WHERE level = 'error'
GROUP BY minute
ORDER BY minute DESC
```

## Limitaciones

WebAssembly tiene algunas limitaciones que conviene tener en cuenta al usar el motor SQL:

- El motor SQL solo admite datos estructurados, como JSON
- El motor SQL solo puede ejecutar consultas en el navegador. Es decir, no puedes lanzar consultas que necesiten acceder a recursos o bases de datos externas
- El motor SQL puede usar como máximo 4 GB de memoria. Si te quedas sin memoria, tendrás que recargar la página para liberarla
