---
title: Utilisation du disque par conteneur
sourceHash: db32da6a38bf
---

# Utilisation du disque par conteneur

La colonne **Disque** de la liste des conteneurs indique combien chaque conteneur a écrit dans son propre système de fichiers, qu'il tourne ou soit arrêté. C'est la même valeur que la première de la colonne `SIZE` de `docker ps -s`, affichée en unités binaires, si bien que le `20.5kB` de Docker apparaît ici comme `20 KB`.

## Ce qui est compté

Docker appelle cela la couche inscriptible du conteneur : tout fichier qu'un conteneur crée ou modifie en dehors de ses montages. Un conteneur qui écrit des fichiers temporaires, des caches ou des logs dans son propre système de fichiers grossit ici, tout comme un conteneur qui lance `apt install` après son démarrage.

Ne sont pas comptés :

- **Les volumes**, nommés ou anonymes
- **Les bind mounts**, comme `./data:/var/lib/postgresql/data`
- **L'image** sur laquelle tourne le conteneur, partagée par tous les conteneurs de cette image
- **Le fichier de log de Docker** pour ce conteneur

La plupart des bases de données et des stockages de logs gardent leurs données dans un volume ou un bind mount, si bien qu'un Postgres contenant 30 Go peut n'afficher ici que quelques Ko. C'est normal : les 30 Go sont dans le montage, pas dans le conteneur.

## Trouver le reste

Docker sait mesurer les volumes, mais ne sait rien du contenu d'un dossier de l'hôte monté en bind mount. Le mesurer oblige à parcourir chacun de ses fichiers, ce que Dozzle ne fait pas. Pour voir où part le reste de l'espace, lancez ceci sur l'hôte :

```sh
# volumes, avec le nombre de conteneurs qui utilisent chacun
docker system df -v

# un dossier monté en bind mount
du -sh /data/postgres
```

Pour surveiller le disque sur lequel se trouvent ces dossiers, montez-le sur la carte de l'hôte comme décrit dans [Métriques de l'hôte](/fr/guide/host-metrics#plus-de-disques).

## Quand la valeur est mise à jour

Docker ne conserve cette valeur nulle part. Il la calcule en parcourant la couche à chaque demande, donc Dozzle la demande rarement :

- une fois pour chaque conteneur, peu après le démarrage de Dozzle
- une fois de plus quand un conteneur s'arrête, puisque sa couche ne peut plus changer ensuite
- pour un conteneur en cours d'exécution, après qu'il a écrit environ 100 Mo, ou toutes les 5 minutes s'il a écrit quoi que ce soit

Les vérifications des conteneurs en cours d'exécution n'ont lieu que tant que quelqu'un a Dozzle ouvert. Un conteneur affiche `–` jusqu'à sa première mesure.

## Limites

- Kubernetes n'a pas de couche inscriptible à remonter, donc la colonne est masquée en [mode k8s](/fr/guide/k8s).
- Un [agent](/fr/guide/agent) mesure ses propres conteneurs. Un agent plus ancien que le Dozzle auquel il se rapporte n'envoie pas de taille, et ses conteneurs affichent `–`.
