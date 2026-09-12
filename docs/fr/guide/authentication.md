---
title: Authentification
sourceHash: 111fbadf1b7a
---

# Authentification

Dozzle prend en charge deux configurations pour l'authentification. Dans la première, vous apportez votre propre méthode d'authentification en protégeant Dozzle derrière un proxy. Dozzle sait lire les en-têtes appropriés sans configuration supplémentaire.

Si vous n'avez pas de solution d'authentification, Dozzle propose une gestion d'utilisateurs simple, basée sur un fichier. Les fournisseurs d'authentification se configurent avec le flag `--auth-provider`. Dans les deux configurations, Dozzle essaie d'enregistrer les paramètres utilisateur sur le disque. Ces données sont écrites dans `/data`.

## <Icon icon="mdi:shield-alert-outline" inline /> Considérations de sécurité

Dozzle a accès à `docker.sock`, ce qui équivaut, sauf restriction, à un accès **root sur l'hôte**. Avant d'exposer Dozzle en dehors de votre réseau privé, passez en revue les points suivants :

- **Placez toujours Dozzle derrière une authentification** s'il est joignable depuis Internet. Utilisez `--auth-provider=simple` ou un forward proxy comme Authelia / Authentik / Cloudflare Access.
- **Laissez les [actions](/fr/guide/actions) et l'[accès shell](/fr/guide/shell) désactivés** sauf si vous en avez besoin. Ils permettent de démarrer, arrêter, recréer des conteneurs et d'y exécuter des commandes arbitraires.
- **Restreignez les utilisateurs avec les [rôles](/fr/guide/authentication/simple#definir-des-roles-specifiques-pour-les-utilisateurs) et les [filtres](/fr/guide/authentication/simple#definir-des-filtres-specifiques-pour-les-utilisateurs)** en mode multi-utilisateur. Sans rôles explicites, un utilisateur voit tous les conteneurs auxquels l'instance Dozzle a accès.
- **N'exposez jamais directement le port de Dozzle en mode forward proxy.** Dozzle fait confiance à `Remote-User` sur chaque requête, et quand aucun en-tête de rôles n'est présent, l'utilisateur reçoit tous les rôles. Quiconque peut atteindre le conteneur sans passer par le proxy s'authentifie comme il veut en positionnant un seul en-tête. Publiez uniquement le proxy et gardez Dozzle sur un réseau interne avec `expose` plutôt que `ports`.
- **Terminez le TLS au niveau du reverse proxy**. Voir [Reverse proxy et chemin de base](/fr/guide/changing-base) pour des exemples Nginx / Traefik / Caddy.
- **Restreignez l'accès à `docker.sock` avec un proxy** si vous n'avez pas besoin des actions. Notez qu'un montage en lecture seule (`/var/run/docker.sock:/var/run/docker.sock:ro`) ne limite _pas_ l'API : le flag `:ro` marque seulement le fichier socket en lecture seule sur le disque, alors que les appels API passent normalement par le socket, donc les opérations de création, suppression et mise à jour restent possibles. Pour réellement restreindre les opérations, placez un proxy de socket comme [`tecnativa/docker-socket-proxy`](https://github.com/Tecnativa/docker-socket-proxy) devant le démon.

## <Icon icon="mdi:key-outline" inline /> Choisir une méthode

| Méthode                                                 | Qui gère les utilisateurs    | À utiliser quand                                                                                                                                                    |
| ------------------------------------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [Simple](/fr/guide/authentication/simple)               | Dozzle, dans `users.yml`     | Vous n'avez pas de solution d'authentification et vous voulez que Dozzle gère les connexions.                                                                       |
| [GitHub et OIDC](/fr/guide/authentication/oauth)        | Dozzle, dans `users.yml`     | Vous voulez que les mêmes utilisateurs de `users.yml` se connectent avec GitHub, Google, Keycloak, Pocket ID, Zitadel ou Authentik plutôt qu'avec un mot de passe.  |
| [OpenID Connect](/fr/guide/authentication/oidc)         | Votre fournisseur d'identité | Vous voulez que Keycloak, Authentik, Zitadel ou Pocket ID possède la liste des utilisateurs, avec les rôles et les filtres lus depuis le token et sans `users.yml`. |
| [Forward proxy](/fr/guide/authentication/forward-proxy) | Votre proxy                  | Vous utilisez déjà Authelia, Authentik, Cloudflare Access ou équivalent, et vous voulez qu'il gère entièrement l'authentification.                                  |

Simple et OAuth sont le même fournisseur : `users.yml` est la liste des utilisateurs dans les deux cas, et OAuth ajoute seulement une deuxième façon de prouver que vous êtes l'un de ces utilisateurs. OpenID Connect et le forward proxy sont ceux qui se distinguent, avec quelque chose d'extérieur à Dozzle qui possède les utilisateurs. Choisissez `oidc` quand votre fournisseur d'identité peut mettre des rôles dans le token, et le forward proxy quand vous avez besoin de règles d'accès à l'échelle d'une organisation ou d'un domaine appliquées devant Dozzle, ce que `users.yml` ne fait volontairement pas.

## <Icon icon="mdi:file-document-edit-outline" inline /> Générer users.yml

Dozzle intègre une commande `generate` pour produire `users.yml`. Voici un exemple :

```sh
docker run -it --rm amir20/dozzle generate admin --password password --email test@email.net --name "John Doe" --user-filter name=foo --user-roles shell > users.yml
```

Dans cet exemple, `admin` est le nom d'utilisateur. L'email et le nom sont optionnels mais recommandés pour afficher les bons avatars. `docker run -it --rm amir20/dozzle generate --help` affiche toutes les options. Le flag `--user-filter` prend une liste de filtres séparés par des virgules. Le flag `--user-roles` prend une liste de rôles séparés par des virgules.

Si vous omettez `--password`, Dozzle le demande sur stdin, pour que le mot de passe ne finisse pas dans l'historique de votre shell. Cela nécessite un terminal interactif, gardez donc les flags `-it` :

```sh
docker run -it --rm amir20/dozzle generate admin --email test@email.net --name "John Doe" > users.yml
```

L'invite est écrite sur stderr, la redirection de stdout vers `users.yml` fonctionne donc toujours. Vous pouvez aussi transmettre le mot de passe par un tube, par exemple `echo "$PASSWORD" | docker run -i --rm amir20/dozzle generate admin > users.yml`.
