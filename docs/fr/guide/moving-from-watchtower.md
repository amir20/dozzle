---
title: Passer de Watchtower à Dozzle
sourceHash: acec3dd8eb61
---

# Passer de Watchtower à Dozzle

Dozzle sait faire ce que fait Watchtower : vérifier selon un planning si vos conteneurs ont des images plus récentes et les mettre à jour. Chaque mise à jour est surveillée, et annulée si le nouveau conteneur ne reste pas en marche. Cette page fait correspondre les réglages de Watchtower à ceux de Dozzle.

## Activer

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

Vous pouvez aussi omettre les deux lignes `DOZZLE_AUTO_UPDATE` et régler le planning dans **Paramètres → Mises à jour**. Arrêtez ensuite Watchtower, pour que les deux ne mettent pas à jour les mêmes conteneurs.

## Réglages

| Watchtower                                                         | Dozzle                                                                                                                                                                                                                             |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--schedule` ou `--interval`                                       | `DOZZLE_AUTO_UPDATE` (`daily`, ou `weekly` le dimanche) et `DOZZLE_AUTO_UPDATE_TIME`. Une fois par jour au plus                                                                                                                    |
| Tous les conteneurs (par défaut)                                   | **Quels conteneurs : Tout**                                                                                                                                                                                                        |
| `--label-enable` avec `com.centurylinklabs.watchtower.enable=true` | **Quels conteneurs : Dozzle et les conteneurs que je choisis** (par défaut) avec `dev.dozzle.update: auto`                                                                                                                         |
| `com.centurylinklabs.watchtower.enable=false`                      | `dev.dozzle.update: off`, ou **Manuel** pour garder la vérification                                                                                                                                                                |
| `--cleanup`                                                        | Toujours actif. L'ancienne image sans tag est supprimée après une mise à jour, et une image précédente est gardée pour revenir en arrière                                                                                          |
| `--monitor-only`                                                   | **Manuel** : le conteneur est vérifié et affiché comme mise à jour, mais jamais mis à jour tout seul                                                                                                                               |
| `--rolling-restart`                                                | Toujours : chaque hôte met à jour un conteneur à la fois                                                                                                                                                                           |
| `--notification-url`                                               | [Alertes et webhooks](/fr/guide/alerts-and-webhooks) sur les événements des conteneurs, ou [Dozzle Cloud](/fr/guide/dozzle-cloud), qui surveille chaque mise à jour et vous prévient quand une nouvelle version commence à échouer |
| `--run-once`                                                       | **Mettre à jour** dans le panneau des mises à jour                                                                                                                                                                                 |
| Identifiants de registre privé                                     | Non pris en charge. Les conteneurs d'un registre privé sont ignorés                                                                                                                                                                |

## Labels

Les labels de Watchtower ne sont pas lus. Remplacez `com.centurylinklabs.watchtower.enable` par `dev.dozzle.update` :

| `dev.dozzle.update` | Ce qui se passe                                                   |
| ------------------- | ----------------------------------------------------------------- |
| `auto`              | Mis à jour selon le planning                                      |
| _(aucun label)_     | Manuel : vérifié et affiché comme mise à jour, que vous appliquez |
| `off`               | Jamais vérifié, jamais mis à jour                                 |

Un label l'emporte sur un choix fait dans l'interface. Les anciens labels de Dozzle sont toujours acceptés : `dev.dozzle.auto-update=true` vaut `auto`, et `dev.dozzle.update-check=false` vaut `off`.

## Ce qui change

- L'ancien conteneur est gardé jusqu'à ce que le nouveau reste en marche, et en bonne santé s'il a un healthcheck. Sinon, l'ancien est remis en place.
- Les conteneurs en mauvaise santé sont ignorés.
- Un conteneur que quelqu'un a [ramené en arrière](/fr/guide/actions#rolling-back) n'est pas remis sur la même image tant qu'une plus récente n'est pas publiée.
- Les images épinglées sur un digest et les images construites localement sont ignorées, faute de version plus récente à comparer.
