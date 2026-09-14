---
title: Assistant de configuration
sourceHash: 7c37c945db57
---

# Assistant de configuration

<Badge type="warning" text="Docker Only" />

Une nouvelle installation de Dozzle s'ouvre sur un court assistant de configuration. Il vous guide à travers les quelques réglages que la plupart des gens modifient juste après l'installation : activer la connexion, autoriser les actions sur les conteneurs et l'accès shell, et connecter Dozzle Cloud. Tout ce qu'il enregistre peut aussi être défini par des flags ou des variables d'environnement, l'assistant est donc facultatif.

L'assistant n'apparaît que sur une nouvelle installation en mode serveur. Les déploiements Swarm et Kubernetes ne l'affichent jamais. Vous pouvez le rouvrir plus tard depuis les paramètres.

## <Icon icon="mdi:format-list-numbered" inline /> Étapes

### 1. Connexion

La connexion vient en premier, pour que rien d'autre ne puisse être modifié sur une instance accessible à tous.

L'assistant vérifie d'abord que `/data` est monté sur un volume. Les paramètres et les utilisateurs y sont écrits, et sans volume ils disparaîtraient à la prochaine recréation du conteneur. Si `/data` n'est pas persistant, l'assistant montre comment le monter et attend que vous cliquiez sur **Vérifier à nouveau**.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

Une fois `/data` persistant, choisissez l'une des trois options :

- **Compte Dozzle** crée un seul utilisateur avec un nom d'utilisateur, un e-mail facultatif et un mot de passe. Dozzle écrit `/data/users.yml` et définit `authProvider: simple`. Consultez [Authentification simple](/fr/guide/authentication/simple) pour ajouter d'autres utilisateurs ou rôles plus tard.
- **Mon proxy** est destiné à Authelia, Authentik, Cloudflare Access et équivalents. Dozzle fait confiance à l'en-tête `Remote-User`, publiez donc uniquement le proxy et jamais le port de Dozzle lui-même. Cela définit `authProvider: forward-proxy`. Consultez [Proxy d'authentification](/fr/guide/authentication/forward-proxy).
- **OIDC** affiche un lien vers le guide [OpenID Connect](/fr/guide/authentication/oidc) et les variables d'environnement à ajouter. OIDC nécessite un secret client, donc rien n'est écrit ici et vous le configurez vous-même.

Si Dozzle n'est accessible que sur votre propre réseau, **Continuer sans connexion** permet de passer cette étape.

Une fois un compte ou un proxy enregistré, Dozzle redémarre immédiatement pour que la connexion soit active avant toute autre modification. Vous arrivez sur la page de connexion, et l'assistant reprend à l'étape suivante une fois connecté.

### 2. Actions et shell

Deux interrupteurs définissent ce que Dozzle a le droit de faire à vos conteneurs :

- **Démarrer, arrêter et redémarrer** active les [actions sur les conteneurs](/fr/guide/actions) (`enableActions`).
- **Shell** active la possibilité de [s'attacher et d'exécuter des commandes](/fr/guide/shell) dans les conteneurs (`enableShell`). Il est désactivé par défaut. Un accès shell à un conteneur vaut souvent un accès à l'hôte, ne l'activez donc que si vous en avez besoin.

Si un réglage est déjà fixé par un flag ou une variable d'environnement, son interrupteur est en lecture seule et l'indique. Comme la connexion, ces interrupteurs ont besoin de `/data` sur un volume et restent en lecture seule tant que ce n'est pas le cas.

### 3. Dozzle Cloud

[Dozzle Cloud](/fr/guide/dozzle-cloud) envoie des alertes dès que quelque chose casse, un résumé matinal de ce qu'il faut corriger, et conserve un historique qui survit aux redémarrages. **Connecter Dozzle Cloud** relie cette instance, et **Pas maintenant** continue. Cette étape est ignorée si l'instance est déjà reliée ou si vous n'avez pas le droit de la relier.

### 4. Redémarrage

La dernière étape liste les modifications enregistrées mais pas encore actives. **Redémarrer Dozzle** redémarre le conteneur, attend qu'il soit de retour et recharge la page. S'il n'y a rien en attente, l'étape indique simplement que vous avez terminé.

Si Dozzle ne peut pas redémarrer tout seul (par exemple s'il ne trouve pas son propre conteneur), l'assistant affiche à la place les variables d'environnement à ajouter à votre fichier compose.

## <Icon icon="mdi:file-cog-outline" inline /> Où les paramètres sont enregistrés

L'assistant enregistre vos choix dans `/data/dozzle.yml`. Dozzle lit ce fichier une seule fois au démarrage, c'est pourquoi les modifications nécessitent un redémarrage. Dozzle redémarre tout seul depuis l'assistant, vous n'avez donc pas à le faire à la main.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
```

| Clé             | Valeurs                           | Équivalent à            |
| --------------- | --------------------------------- | ----------------------- |
| `authProvider`  | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`  |
| `enableActions` | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS` |
| `enableShell`   | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`   |

Les flags et les variables d'environnement l'emportent toujours sur le fichier. Si `DOZZLE_ENABLE_ACTIONS` est défini, la valeur de `dozzle.yml` est ignorée et l'assistant affiche l'interrupteur comme verrouillé. Pour gérer à nouveau un réglage depuis l'assistant, retirez la variable de votre fichier compose.

## <Icon icon="mdi:shield-lock-outline" inline /> Sécurité

- **La connexion est la première étape.** Un redémarrage après l'enregistrement d'un compte ou d'un proxy active la connexion avant que tout autre réglage puisse être modifié.
- **Seul un utilisateur connecté peut modifier les actions et le shell ou redémarrer Dozzle.** L'utilisateur doit avoir tous les rôles.
- **Sans connexion, seule une nouvelle installation a une fenêtre de 15 minutes.** Quand `authProvider` vaut `none`, ces réglages ne peuvent être modifiés que dans les 15 minutes qui suivent le premier démarrage d'une nouvelle installation, c'est-à-dire dont `/data` était vide. Une installation qui a déjà des données de démarrages précédents n'a jamais cette fenêtre, un redémarrage de l'hôte ou une mise à jour de l'image ne peut donc pas l'ouvrir. En dehors de la fenêtre, utilisez les variables d'environnement ou activez la connexion.
- **Les routes sont toujours décidées au démarrage.** L'assistant écrit uniquement dans `dozzle.yml`. Les endpoints des actions et du shell sont enregistrés au démarrage de Dozzle, exactement comme avec les variables d'environnement, donc rien n'est activé tant que Dozzle n'a pas redémarré.
