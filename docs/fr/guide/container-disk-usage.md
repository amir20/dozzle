---
title: Utilisation du disque par conteneur
sourceHash: 36536624d20d
---

# Utilisation du disque par conteneur

La colonne **Disque** de la liste des conteneurs indique l'espace disque occupé par chaque conteneur, qu'il tourne ou soit arrêté. Elle additionne deux choses que Docker sait mesurer : ce que le conteneur a écrit dans son propre système de fichiers, et les volumes Docker qu'il monte. Survolez la valeur pour voir chaque partie.

Les tailles sont en unités binaires, si bien que le `20.5kB` de Docker apparaît ici comme `20 KB`.

## Ce qui est compté

- **La couche inscriptible** : tout fichier qu'un conteneur crée ou modifie en dehors de ses montages, comme des fichiers temporaires, des caches ou des paquets installés après son démarrage. C'est la première valeur de la colonne `SIZE` de `docker ps -s`.
- **Les volumes**, nommés ou anonymes, que le conteneur monte. Ce sont les tailles que liste `docker system df -v`. Un volume utilisé par plusieurs conteneurs compte pour chacun d'eux, et l'info-bulle indique qu'il est partagé.

Ne sont pas comptés :

- **Les bind mounts**, comme `./data:/var/lib/postgresql/data`
- **L'image** sur laquelle tourne le conteneur, partagée par tous les conteneurs de cette image
- **Le fichier de log de Docker** pour ce conteneur
- **Les volumes qu'aucun conteneur n'utilise**, puisqu'ils n'ont pas de ligne où apparaître. Ils comptent à la place dans l'[espace récupérable](#espace-recuperable) de l'hôte.

## Bind mounts

Docker ne sait rien du contenu d'un dossier de l'hôte monté en bind mount, et le mesurer oblige à parcourir chacun de ses fichiers, ce que Dozzle ne fait pas. Une base de données qui garde ses données dans un bind mount affiche ici quelques Ko même si elle contient plusieurs Go. Pour en mesurer un, lancez ceci sur l'hôte :

```sh
du -sh /data/postgres
```

Pour surveiller le disque sur lequel se trouvent ces dossiers, montez-le sur la carte de l'hôte comme décrit dans [Métriques de l'hôte](/fr/guide/host-metrics#plus-de-disques).

## Espace récupérable

La carte de l'hôte affiche **Récupérable** à côté de son indicateur de disque : l'espace occupé par ce qu'aucun conteneur n'utilise, le même total que la colonne `RECLAIMABLE` de `docker system df`. Survolez-le pour voir la part des images inutilisées, des volumes inutilisés, des conteneurs arrêtés et du cache de build. Il est actualisé en même temps que les volumes, donc au plus toutes les 20 minutes.

## Quand la valeur est mise à jour

Docker ne conserve ces valeurs nulle part. Il les calcule en parcourant les fichiers à chaque demande, donc Dozzle les demande rarement.

La couche inscriptible est mesurée :

- une fois pour chaque conteneur, peu après le démarrage de Dozzle
- une fois de plus quand un conteneur s'arrête, puisque sa couche ne peut plus changer ensuite
- pour un conteneur en cours d'exécution, après qu'il a écrit environ 100 Mo, ou toutes les 5 minutes s'il a écrit quoi que ce soit

Les volumes sont mesurés juste après ce premier passage, puis au plus toutes les 20 minutes. Docker ne sait mesurer que tous les volumes à la fois, donc chaque actualisation parcourt chaque volume de l'hôte. Un conteneur créé entre-temps affiche ses volumes à l'actualisation suivante.

Tout ce qui suit le premier passage n'a lieu que tant que quelqu'un a Dozzle ouvert. Un conteneur affiche `–` jusqu'à sa première mesure.

## Limites

- Kubernetes n'a ni couche inscriptible ni volumes Docker à remonter, donc la colonne est masquée en [mode k8s](/fr/guide/k8s).
- Les volumes d'un pilote de plugin ne savent généralement pas indiquer leur taille et sont ignorés.
- Un [agent](/fr/guide/agent) mesure ses propres conteneurs. Un agent plus ancien que le Dozzle auquel il se rapporte n'envoie pas de tailles, et ses conteneurs affichent `–`.
