---
title: Passer de Watchtower à Dozzle
sourceHash: 3c48dbce256f
---

# Passer de Watchtower à Dozzle

Dozzle sait faire ce que fait Watchtower : vérifier selon un planning si vos conteneurs ont des images plus récentes et les mettre à jour. Chaque mise à jour est surveillée, et si le nouveau conteneur ne reste pas en marche, l'ancien est remis en place. Cette page fait correspondre les réglages de Watchtower à ceux de Dozzle.

## L'activer

Dozzle a besoin des actions et de `/data` sur un volume :

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle-data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_ENABLE_ACTIONS: true
      DOZZLE_AUTO_UPDATE: daily
      DOZZLE_AUTO_UPDATE_TIME: "04:00"
volumes:
  dozzle-data:
```

Vous pouvez omettre les deux lignes `DOZZLE_AUTO_UPDATE` et régler le planning dans **Paramètres → Mises à jour** à la place. Choisissez aussi **Quels conteneurs** à cet endroit. Arrêtez ensuite Watchtower, pour que les deux ne mettent pas à jour les mêmes conteneurs.

## Réglages

| Watchtower                                                         | Dozzle                                                                                                                                                                                                                            |
| ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` ou `--interval`                                       | `DOZZLE_AUTO_UPDATE` (`daily`, ou `weekly` le dimanche) et `DOZZLE_AUTO_UPDATE_TIME`. Au plus une fois par jour                                                                                                                   |
| Tous les conteneurs (par défaut)                                   | **Quels conteneurs : Tout**                                                                                                                                                                                                       |
| `--label-enable` avec `com.centurylinklabs.watchtower.enable=true` | **Quels conteneurs : Conteneurs avec label** (par défaut) avec `dev.dozzle.update: auto`                                                                                                                                          |
| `com.centurylinklabs.watchtower.enable=false`                      | `dev.dozzle.update: off`                                                                                                                                                                                                          |
| `--cleanup`                                                        | Toujours actif. L'ancienne image sans tag est supprimée après une mise à jour, et une image précédente est gardée pour revenir en arrière                                                                                         |
| `--monitor-only`                                                   | Laisser le conteneur sans label avec **Conteneurs avec label** : il est vérifié et affiché comme mise à jour, mais jamais mis à jour tout seul                                                                                    |
| `--rolling-restart`                                                | Toujours : chaque hôte met à jour un conteneur à la fois                                                                                                                                                                          |
| `--notification-url`                                               | [Alertes et webhooks](/fr/guide/alerts-and-webhooks) sur les évènements de conteneurs, ou [Dozzle Cloud](/fr/guide/dozzle-cloud), qui surveille chaque mise à jour et vous prévient quand une nouvelle version commence à échouer |
| `--run-once`                                                       | **Mettre à jour** dans le panneau des mises à jour                                                                                                                                                                                |
| Identifiants de registre privé                                     | Non pris en charge. Les conteneurs d'un registre privé sont ignorés                                                                                                                                                               |

## Labels

Les labels de Watchtower ne sont pas lus. Remplacez `com.centurylinklabs.watchtower.enable` par `dev.dozzle.update` :

| `dev.dozzle.update` | Ce qui se passe                                                                                                      |
| ------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `auto`              | Mis à jour selon le planning, sauf si **Quels conteneurs** vaut **Dozzle seulement**                                 |
| _(aucun label)_     | Mis à jour selon le planning avec **Tout**. Sinon vérifié et affiché comme mise à jour, que vous appliquez vous-même |
| `off`               | Jamais vérifié, jamais mis à jour                                                                                    |

Les anciens labels de Dozzle sont toujours acceptés : `dev.dozzle.auto-update=true` est lu comme `auto`, et `dev.dozzle.update-check=false` comme `off`.

## Ce qui change

- L'ancien conteneur est gardé jusqu'à ce que le nouveau reste en marche, et en bonne santé s'il a un healthcheck. Sinon, l'ancien est remis en place.
- Les conteneurs en mauvaise santé sont ignorés.
- Les conteneurs arrêtés ne sont jamais mis à jour, même avec le label `auto`.
- Avec [Dozzle Cloud](/fr/guide/dozzle-cloud), une mise à jour planifiée qui commence à échouer peut être [annulée](/fr/guide/actions#rolling-back). Le planning laisse alors ce conteneur de côté jusqu'à la publication d'une image plus récente.
- Les images figées sur un digest et les images construites localement sont ignorées, faute de version plus récente à comparer.
