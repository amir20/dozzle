---
title: Alertes et webhooks
sourceHash: daf372975955
---

# Alertes et webhooks

Dozzle dispose d'un système d'alertes qui permet de surveiller les logs des conteneurs, les métriques de ressources et les évènements de cycle de vie, et de recevoir des notifications quand certaines conditions sont remplies. Les alertes utilisent des expressions personnalisables pour filtrer les conteneurs et déclencher les conditions, et peuvent envoyer des notifications vers des webhooks, Slack, Discord, ntfy ou [Dozzle Cloud](/fr/guide/dozzle-cloud).

## <Icon icon="mdi:format-list-bulleted-type" inline /> Types d'alertes

Dozzle prend en charge trois types d'alertes, toutes configurées de la même façon depuis la page **Notifications** :

| Type                           | Se déclenche sur                                   | Exemple d'usage                 |
| ------------------------------ | -------------------------------------------------- | ------------------------------- |
| [**Log**](#log-alerts)         | Un message de log correspondant à un motif         | Erreurs 5xx, traces d'exécution |
| [**Métrique**](#metric-alerts) | Le CPU ou la mémoire franchissant un seuil         | Conteneur dépassant 90 % de CPU |
| [**Évènement**](#event-alerts) | Les évènements de cycle de vie remontés par Docker | Kills OOM, conteneurs non sains |

Chaque alerte associe une **expression de conteneur** (quels conteneurs surveiller) à une **expression de déclenchement** (la condition qui déclenche).

> [!IMPORTANT]
> Les configurations d'alertes et de destinations sont stockées dans le dossier `/data`. Vous devez monter ce dossier en volume pour conserver vos paramètres de notification entre les redémarrages du conteneur.

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/data:/data -p 8080:8080 amir20/dozzle:latest
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/data:/data
    ports:
      - 8080:8080
```

:::

## <Icon icon="mdi:send-outline" inline /> Configurer une destination

Avant de créer des alertes, vous devez configurer au moins une destination de notification. Allez sur la page **Notifications** de Dozzle et cliquez sur **Ajouter une destination**.

### Webhook

Les webhooks envoient une requête HTTP POST vers l'URL de votre choix. Dozzle inclut des modèles de payload intégrés pour les services les plus courants :

- **Slack**, mis en forme avec des blocs et du markdown
- **Discord**, mis en forme pour l'API webhook de Discord
- **ntfy**, mis en forme pour les notifications push de [ntfy.sh](https://ntfy.sh)
- **Custom**, un payload JSON générique que vous pouvez adapter

Vous pouvez aussi écrire votre propre modèle de payload avec la syntaxe `text/template` de Go. Les variables suivantes sont disponibles :

<div v-pre>

| Variable                  | Description                                     |
| ------------------------- | ----------------------------------------------- |
| `{{.Detail}}`             | Résumé (message de log ou valeurs de métriques) |
| `{{.Container.Name}}`     | Nom du conteneur                                |
| `{{.Container.Image}}`    | Image du conteneur                              |
| `{{.Container.HostName}}` | Nom de l'hôte Docker                            |
| `{{.Container.State}}`    | État du conteneur                               |
| `{{.Log.Message}}`        | Contenu du message de log                       |
| `{{.Log.Level}}`          | Niveau de log                                   |
| `{{.Log.Timestamp}}`      | Horodatage du log                               |
| `{{.Log.Stream}}`         | Type de flux (stdout/stderr)                    |
| `{{.Stat.CPUPercent}}`    | Pourcentage d'utilisation CPU                   |
| `{{.Stat.MemoryPercent}}` | Pourcentage d'utilisation mémoire               |
| `{{.Stat.MemoryUsage}}`   | Utilisation mémoire en octets                   |
| `{{.Subscription.Name}}`  | Nom de la règle d'alerte                        |

</div>

> [!TIP]
> Utilisez le bouton **Test** pour vérifier que votre webhook fonctionne avant d'enregistrer.

### Dozzle Cloud

Vous pouvez aussi envoyer les alertes vers [Dozzle Cloud](/fr/guide/dozzle-cloud) pour une supervision centralisée de plusieurs instances Dozzle. Voir le [guide Dozzle Cloud](/fr/guide/dozzle-cloud) pour plus de détails.

## <Icon icon="mdi:plus-circle-outline" inline /> Créer une alerte

Allez sur la page **Notifications** et cliquez sur **Ajouter une alerte**. Chaque alerte a une **expression de conteneur** plus une expression de déclenchement de type **log**, **métrique** ou **évènement**.

### Expression de conteneur

L'expression de conteneur sélectionne les conteneurs à surveiller. Propriétés disponibles :

| Propriété  | Type   | Exemple                         |
| ---------- | ------ | ------------------------------- |
| `name`     | chaîne | `name contains "api"`           |
| `image`    | chaîne | `image == "nginx:latest"`       |
| `state`    | chaîne | `state == "running"`            |
| `health`   | chaîne | `health == "unhealthy"`         |
| `hostName` | chaîne | `hostName == "prod-host"`       |
| `labels`   | map    | `labels["env"] == "production"` |

Vous pouvez combiner les conditions avec `&&` (ET), `||` (OU) et `!` (NON) :

```
name contains "api" && labels["env"] == "production"
```

## <Icon icon="mdi:text-search" inline /> Alertes de log

### Expression de log

L'expression de log filtre les messages de log qui déclenchent l'alerte. Propriétés disponibles :

| Propriété | Type       | Exemple                    |
| --------- | ---------- | -------------------------- |
| `message` | chaîne/map | `message contains "error"` |
| `level`   | chaîne     | `level == "error"`         |
| `stream`  | chaîne     | `stream == "stderr"`       |
| `type`    | chaîne     | `type == "complex"`        |

Pour les logs JSON, vous pouvez accéder aux champs imbriqués avec la notation par points :

```
message.status >= 500 && message.path contains "/api"
```

Les opérateurs de chaîne pris en charge incluent `contains`, `startsWith`, `endsWith` et `matches` (regex).

### Exemples de logs

**Alerter sur toutes les erreurs des conteneurs de production :**

```
Container: labels["env"] == "production"
Log:       level == "error"
```

**Alerter sur les erreurs HTTP 5xx des conteneurs API :**

```
Container: name contains "api"
Log:       message.status >= 500
```

**Alerter sur toute sortie stderr d'une image précise :**

```
Container: image startsWith "myapp/"
Log:       stream == "stderr"
```

**Alerter sur les réponses lentes de l'API en production :**

```
Container: name contains "api" && labels["env"] == "production"
Log:       message.duration > 5000 && message.path contains "/api"
```

**Alerter sur les échecs d'authentification avec une regex :**

```
Container: name contains "auth" || name contains "gateway"
Log:       message matches "(?i)(unauthorized|forbidden|invalid token)"
```

> [!NOTE]
> L'éditeur d'alertes propose l'autocomplétion et une validation en temps réel. Vous pouvez prévisualiser les conteneurs et les logs correspondants avant d'enregistrer.

## <Icon icon="mdi:chart-line" inline /> Alertes de métriques

Les alertes de métriques se déclenchent quand l'utilisation CPU ou mémoire d'un conteneur franchit un seuil. L'expression de déclenchement est évaluée sur une moyenne lissée des statistiques échantillonnées sur une fenêtre glissante, ce qui évite les fausses alertes dues à de courts pics.

### Expression de métrique

Propriétés disponibles :

| Propriété     | Type   | Description                                                    |
| ------------- | ------ | -------------------------------------------------------------- |
| `cpu`         | nombre | Pourcentage d'utilisation CPU (0–100), identique à l'interface |
| `memory`      | nombre | Pourcentage d'utilisation mémoire (0–100)                      |
| `memoryUsage` | nombre | Utilisation mémoire en octets                                  |

### Temporisation et fenêtre d'échantillonnage

- **Fenêtre d'échantillonnage** : le nombre de secondes de statistiques moyennées avant l'évaluation de l'expression. Les fenêtres longues lissent les pics, les courtes réagissent plus vite.
- **Temporisation** : le nombre minimum de secondes entre deux déclenchements consécutifs pour le même conteneur. Évite les avalanches d'alertes quand un conteneur reste au-dessus du seuil.

### Exemples de métriques

**CPU élevé sur les conteneurs de production :**

```
Container: labels["env"] == "production"
Metric:    cpu > 90
```

**Pression mémoire sur un service précis :**

```
Container: name contains "api"
Metric:    memory > 85
```

**Utilisation mémoire absolue (1 Gio) :**

```
Container: name == "postgres"
Metric:    memoryUsage > 1073741824
```

## <Icon icon="mdi:bell-outline" inline /> Alertes d'évènements

Les alertes d'évènements se déclenchent sur les évènements de cycle de vie des conteneurs Docker, ce qui est pratique pour repérer les plantages, les kills OOM et les changements d'état de santé sans analyser les logs.

### Expression d'évènement

Propriétés disponibles :

| Propriété    | Type       | Description                                                   |
| ------------ | ---------- | ------------------------------------------------------------- |
| `name`       | chaîne     | Nom de l'évènement (voir plus bas)                            |
| `actorId`    | chaîne     | Identifiant de l'acteur Docker (en général l'id du conteneur) |
| `attributes` | map        | Attributs de l'évènement Docker (variables selon le type)     |
| `timestamp`  | date/heure | Moment où l'évènement s'est produit                           |

Les noms d'évènements Docker courants sont `start`, `stop`, `die`, `kill`, `oom`, `restart`, `destroy` et `health_status`.

Pour les évènements `health_status`, Dozzle expose l'état courant dans `attributes["healthStatus"]` (`healthy` ou `unhealthy`).

### Exemples d'évènements

**Alerter quand un conteneur de production s'arrête :**

```
Container: labels["env"] == "production"
Event:     name == "die"
```

**Alerter sur les kills OOM :**

```
Container: true
Event:     name == "oom"
```

**Alerter quand un conteneur devient non sain :**

```
Container: true
Event:     name == "health_status" && attributes["healthStatus"] == "unhealthy"
```

**Alerter sur les sorties inattendues (en ignorant les arrêts propres) :**

Les codes de sortie 0 (succès), 130 (SIGINT), 143 (SIGTERM) et 137 (SIGKILL) surviennent lors d'un `docker stop`, d'un Ctrl+C ou d'un cycle de mise à jour, ils sont donc exclus pour éviter le bruit. Les vraies sorties en erreur (1, 2, 125, ...) déclenchent toujours l'alerte.

```
Container: name contains "worker"
Event:     name == "die" && !(attributes["exitCode"] in ["0", "130", "143", "137"])
```

## <Icon icon="mdi:cog-outline" inline /> Gérer les alertes

Depuis la page Notifications, vous pouvez :

- **Activer ou désactiver** des alertes sans les supprimer
- **Modifier** les expressions et les destinations d'une alerte
- **Consulter les statistiques**, dont le nombre de déclenchements, les conteneurs correspondants et la date du dernier déclenchement
- **Supprimer** les alertes devenues inutiles
