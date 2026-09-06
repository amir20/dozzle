---
title: Perfil por defecto
sourceHash: 1ef0edd24fb4
---

# Perfil por defecto

Dozzle guarda en disco las preferencias de interfaz de cada usuario (tema, idioma, contenedores fijados, grupos plegados, claves JSON visibles, etc.) en `/data/<username>/profile.json`. Cuando la [autenticación](/es/guide/authentication) está desactivada, o para cualquier usuario que todavía no haya iniciado sesión ni personalizado su configuración, Dozzle recurre a un perfil especial llamado `__default__`.

Puedes distribuir un perfil preconfigurado creando el archivo `/data/__default__/profile.json`. Los visitantes anónimos y los usuarios nuevos sin perfil guardado cargarán esta configuración en su primera visita.

## Ubicación del archivo

```
/data/__default__/profile.json
```

Si el archivo no existe, Dozzle arranca con sus valores por defecto. Solo hace falta crearlo si quieres cambiarlos.

## Ejemplo

```json
{
  "settings": {
    "showTimestamp": true,
    "showStd": false,
    "showAllContainers": false,
    "softWrap": true,
    "collapseNav": false,
    "smallerScrollbars": false,
    "search": false,
    "compact": false,
    "menuWidth": 15,
    "size": "medium",
    "lightTheme": "auto",
    "hourStyle": "auto",
    "dateLocale": "auto",
    "locale": "en",
    "groupContainers": "at-least-2",
    "automaticRedirect": "delayed"
  },
  "pinned": [],
  "visibleKeys": [],
  "collapsedGroups": []
}
```

Todos los campos son opcionales: incluye solo los que quieras sobrescribir.

## Opciones disponibles

| Campo               | Tipo    | Descripción                                                         |
| ------------------- | ------- | ------------------------------------------------------------------- |
| `showTimestamp`     | boolean | Muestra la marca de tiempo junto a cada línea de log                |
| `showStd`           | boolean | Muestra el indicador de flujo stdout/stderr                         |
| `showAllContainers` | boolean | Incluye los contenedores parados en la barra lateral                |
| `softWrap`          | boolean | Ajusta las líneas largas en vez de usar desplazamiento horizontal   |
| `collapseNav`       | boolean | Arranca con la barra lateral plegada                                |
| `smallerScrollbars` | boolean | Usa barras de desplazamiento más finas                              |
| `search`            | boolean | Activa la búsqueda integrada por defecto                            |
| `compact`           | boolean | Espaciado compacto entre líneas de log                              |
| `menuWidth`         | number  | Ancho de la barra lateral en porcentaje de la ventana. Máximo `50`. |
| `size`              | string  | Tamaño de letra: `small`, `medium`, `large`                         |
| `lightTheme`        | string  | Preferencia de tema: `auto`, `light`, `dark`                        |
| `hourStyle`         | string  | Formato de hora: `auto`, `12`, `24`                                 |
| `dateLocale`        | string  | Formato de fecha y hora: `auto`, `en-US`, `en-GB`, `de-DE`, `en-CA` |
| `locale`            | string  | Idioma de la interfaz (por ejemplo `en`, `fr`, `de`)                |
| `groupContainers`   | string  | Agrupación en la barra lateral: `always`, `at-least-2`, `never`     |
| `automaticRedirect` | string  | Redirección a un contenedor nuevo: `instant`, `delayed`, `none`     |

No se admiten valores fuera de estos conjuntos, así que `groupContainers: "stack"` o un `dateLocale` de `fr-FR` no harán lo que esperas.

Los campos de primer nivel `pinned`, `visibleKeys` y `collapsedGroups` aceptan arrays y permiten fijar contenedores o plegar grupos de antemano para quienes entran por primera vez. Dozzle también escribe `releaseSeen`, `dismissedImageUpdates` y `dismissedLinkHint` en el primer nivel para recordar lo que cada usuario ya ha descartado. Poner `dismissedLinkHint: true` oculta para todo el mundo el aviso de enlace que aparece la primera vez.

## Cómo funciona

- Al cargar la página, Dozzle lee `/data/<username>/profile.json` para el usuario que ha iniciado sesión, o `/data/__default__/profile.json` cuando no hay ningún usuario autenticado.
- Cuando un usuario cambia una opción en la interfaz, el nuevo valor se guarda bajo su propio nombre de usuario (o de vuelta en `__default__` si la autenticación está desactivada).
- Por eso el perfil `__default__` es a la vez la **plantilla para los visitantes nuevos** y el **perfil real del usuario anónimo** en despliegues sin autenticación.

::: tip
Si solo quieres sembrar unos valores por defecto pero dejar que el usuario anónimo los cambie en caliente, monta el archivo en modo solo lectura: Dozzle no podrá guardar los cambios pero la interfaz seguirá funcionando.
:::
