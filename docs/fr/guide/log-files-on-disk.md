---
title: Suivre des fichiers de logs sur disque
sourceHash: e6e23f2438f9
---

# Suivre des fichiers de logs sur disque

Certains conteneurs écrivent leurs logs dans des fichiers plutôt que sur `stdout` ou `stderr`. Dozzle ne peut lire que ce que Docker capture lui-même, c'est-à-dire `stdout` et `stderr`, exactement comme `docker logs`. Les fichiers situés dans un conteneur ne sont pas visibles depuis les autres conteneurs, donc Dozzle n'a aucun moyen de les atteindre.

## Écrire dans les flux plutôt que dans un fichier

La meilleure solution est de cesser d'écrire dans des fichiers. La plupart des applications ont une option de configuration pour écrire dans la console, et le [twelve factor app](https://12factor.net/logs) explique pourquoi c'est le bon comportement par défaut.

Si l'application ne peut pas être configurée, créez dans votre `Dockerfile` un lien symbolique du fichier de logs vers la sortie standard du conteneur. C'est ce que fait l'image officielle nginx :

```dockerfile
RUN ln -sf /dev/stdout /var/log/nginx/access.log \
    && ln -sf /dev/stderr /var/log/nginx/error.log
```

## Suivre un fichier avec un sidecar

Quand aucune des deux options n'est possible, lancez un petit conteneur Alpine qui suit le fichier et laisse Docker capturer sa sortie. Dozzle l'affiche alors comme n'importe quel autre conteneur.

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

La version Compose est utile si vous voulez que le flux de logs survive à un redémarrage du serveur. Lors des tests, Alpine consommait environ `~50KB` de mémoire.

### Pourquoi `-F` et pas `-f`

`tail -f` suit le descripteur de fichier ouvert. Quand le fichier est tourné, le descripteur pointe vers l'ancien fichier renommé et le flux se tait. `tail -F` suit le chemin et rouvre le fichier après une rotation, donc il continue de fonctionner.

Pour la même raison, montez le **répertoire** plutôt que le fichier. Un bind mount d'un fichier unique est lié à l'inode de ce fichier : une rotation sur l'hôte remplace le fichier et le conteneur continue de regarder l'ancien, même avec `-F`.

### Amorcer l'historique

Docker ne conserve que ce que le conteneur a affiché depuis son démarrage, donc redémarrer le sidecar efface tout ce que Dozzle avait. `-n 1000` affiche les 1000 dernières lignes au démarrage pour que la vue ne soit pas vide.

### Plusieurs fichiers

`tail` préfixe chaque bloc avec le nom du fichier quand on lui en passe plusieurs. Les globs nécessitent un shell, car l'image n'a pas d'entrypoint pour les développer :

```sh
docker run -d -v /var/log:/logs:ro alpine sh -c 'tail -n 1000 -F /logs/*.log'
```

Le label `dev.dozzle.name` ci-dessus donne au sidecar un nom lisible dans l'interface. Voir [Noms de conteneurs](/fr/guide/container-names) pour plus de détails.
