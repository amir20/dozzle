---
title: Mode agent
sourceHash: 34df9234d941
---

# Mode agent

<Badge type="warning" text="Docker uniquement" />

Dozzle peut fonctionner en mode agent, ce qui permet d'exposer des hôtes Docker à d'autres instances de Dozzle. Toute la communication passe par une connexion sécurisée en TLS. Vous pouvez donc déployer Dozzle sur un hôte distant et vous y connecter depuis votre machine locale.

> [!NOTE] Vous utilisez Docker Swarm ?
> Si vous utilisez le mode Docker Swarm, vous n'avez pas besoin d'agents. Dozzle se découvre automatiquement et crée un cluster en mode swarm. Voir [Mode Swarm](/fr/guide/swarm-mode) pour plus d'informations.

## <Icon icon="mdi:plus-box-outline" inline /> Comment créer un agent

Pour créer un agent Dozzle, lancez Dozzle avec la sous-commande `agent`. Voici un exemple :

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

> [!NOTE] Utilisateurs d'un proxy de socket Docker
> Si vous utilisez un agent distant, vous **NE POUVEZ PAS** ajouter un proxy de socket par-dessus l'agent. Les agents Dozzle **REMPLACENT** l'usage d'un proxy, voir [Hôtes distants](/fr/guide/remote-hosts) pour plus d'informations et pour savoir comment utiliser un proxy de socket à la place d'un agent.

L'agent démarre et écoute sur le port `7007`. Vous pouvez vous y connecter depuis l'interface de Dozzle en fournissant l'adresse IP et le port de l'agent. L'agent n'affiche que les conteneurs disponibles sur l'hôte où il tourne.

> [!TIP]
> Vous n'avez pas besoin d'exposer le port 7007 si vous utilisez un réseau Docker. L'agent est joignable par les autres conteneurs du même réseau.

## <Icon icon="mdi:connection" inline /> Comment se connecter à un agent

Pour vous connecter à un agent, fournissez son adresse IP et son port. Voici un exemple :

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest --remote-agent agent:7007
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent:7007
    ports:
      - 8080:8080 # Port de l'interface Dozzle
```

:::

Notez qu'il n'est pas nécessaire de monter le socket Docker local pour se connecter à des agents, auquel cas l'interface n'affiche que les conteneurs disponibles sur les agents.

> [!TIP]
> Si vous voulez aussi inclure les conteneurs de l'hôte dans l'interface, montez le socket `docker.sock` comme dans l'exemple de [prise en main](/fr/guide/getting-started).

> [!TIP]
> Vous pouvez vous connecter à plusieurs agents en fournissant plusieurs variables d'environnement `DOZZLE_REMOTE_AGENT`. Par exemple, `DOZZLE_REMOTE_AGENT=agent1:7007,agent2:7007`.

## <Icon icon="mdi:group" inline /> Groupes d'hôtes

Quand vous gérez de nombreux agents répartis sur différents environnements, vous pouvez affecter chaque agent à un groupe nommé. Les groupes apparaissent sous forme de sections repliables dans la barre latérale, et chaque groupe dispose d'un bouton « tout fusionner » pour voir les logs combinés de tous les hôtes du groupe.

Le format de la chaîne de connexion est `endpoint|name|group`, les trois parties sont optionnelles :

| Format                          | Résultat                               |
| ------------------------------- | -------------------------------------- |
| `agent:7007`                    | Pas de nom personnalisé, pas de groupe |
| `agent:7007\|web-1`             | Nom personnalisé, pas de groupe        |
| `agent:7007\|web-1\|Production` | Nom personnalisé + groupe              |
| `agent:7007\|\|Production`      | Nom d'hôte par défaut + groupe         |

::: code-group

```sh
docker run -p 8080:8080 amir20/dozzle:latest \
  --remote-agent agent1:7007|web-1|Production \
  --remote-agent agent2:7007|web-2|Production \
  --remote-agent agent3:7007|dev-1|Development
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_REMOTE_AGENT=agent1:7007|web-1|Production,agent2:7007|web-2|Production,agent3:7007|dev-1|Development
    ports:
      - 8080:8080
```

:::

La barre latérale affichera :

```
▾ Production
    web-1
    web-2
▾ Development
    dev-1
  ungrouped-host   ← les agents sans groupe apparaissent en dessous
```

Cliquer sur l'icône de fusion à côté d'un nom de groupe ouvre une vue de logs combinée qui streame depuis tous les hôtes du groupe. Cette vue fusionnée est aussi accessible directement à l'adresse `/host-group/<group-name>`.

Les agents sans groupe continuent de fonctionner exactement comme avant et apparaissent sous les sections groupées.

## <Icon icon="mdi:alert-circle-outline" inline /> Problèmes courants

### Un agent n'apparaît pas

Si vous voyez `An agent with an existing ID was found. Removing the duplicate host.`, c'est que deux hôtes utilisent le même identifiant de serveur.

Dozzle utilise l'API Docker pour collecter des informations sur les hôtes. Chaque agent a besoin d'un identifiant d'hôte unique qui reste le même entre les redémarrages, afin d'être identifié correctement. Actuellement, les agents identifient l'hôte par l'identifiant système de Docker ou par l'identifiant de nœud.

Dans un environnement Swarm, c'est l'identifiant de nœud qui est utilisé. Si vous constatez que tous les hôtes ne sont pas visibles, cela peut venir d'hôtes en double configurés avec le même identifiant d'hôte.

Pour résoudre ce problème, supprimez `/var/lib/docker/engine-id` de votre système et redémarrez. Cela élimine les conflits causés par des identifiants d'hôte en double. Pour plus d'informations et de conseils de dépannage, consultez la [FAQ](/fr/guide/faq#i-am-seeing-duplicate-hosts-error-in-the-logs-how-do-i-fix-it).

## <Icon icon="mdi:cog-outline" inline /> Options avancées

### Configurer un healthcheck

Vous pouvez définir un healthcheck pour l'agent, comme pour l'instance Dozzle principale. En mode agent, le healthcheck vérifie la connexion de l'agent à Docker. Si Docker n'est pas joignable, l'agent est marqué comme non sain et n'apparaît pas dans l'interface.

Pour mettre en place le healthcheck, utilisez la sous-commande `healthcheck`. Voici un exemple :

```yml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    healthcheck:
      test: ["CMD", "/dozzle", "healthcheck"]
      interval: 5s
      retries: 5
      start_period: 5s
      start_interval: 5s
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

### Changer le nom de l'agent

Comme pour une instance Dozzle, vous pouvez changer le nom de l'agent avec la variable d'environnement `DOZZLE_HOSTNAME`. Voici un exemple :

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -p 7007:7007 amir20/dozzle:latest agent --hostname my-special-name
```

```yaml [docker-compose.yml]
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_HOSTNAME=my-special-name
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - 7007:7007
```

:::

Le nom de l'agent devient `my-special-name` et sera repris dans l'interface lors de la connexion à l'agent.

### Configurer des filtres

Vous pouvez configurer des filtres sur l'agent pour limiter les conteneurs auxquels il a accès. Ces filtres sont transmis directement à Docker et restreignent ce que Dozzle peut voir.

```yaml
services:
  dozzle-agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_FILTER=label=color
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

L'agent n'affichera que les conteneurs portant le label `color`. Gardez à l'esprit que ces filtres se combinent avec les filtres de l'interface pour restreindre encore la liste des conteneurs. Pour en savoir plus sur les différents types de filtres, lisez la [documentation des filtres](/fr/guide/filters#ui-agents-and-user-filters).

### Certificats personnalisés

Par défaut, Dozzle utilise des certificats auto-signés pour la communication entre agents. C'est un certificat privé valide uniquement pour d'autres instances de Dozzle. C'est sûr et recommandé dans la plupart des cas. En revanche, si Dozzle est exposé publiquement et qu'un attaquant connaît exactement le port sur lequel tourne l'agent, il peut monter sa propre instance de Dozzle et se connecter à l'agent. Pour éviter cela, vous pouvez fournir vos propres certificats.

Pour fournir des certificats personnalisés, utilisez un montage ou des secrets. Par défaut, Dozzle cherche les certificats dans `/dozzle_cert.pem` et `/dozzle_key.pem`, mais vous pouvez changer ces chemins avec les flags `--cert` et `--key` ou les variables d'environnement `DOZZLE_CERT` et `DOZZLE_KEY`.

Voici un exemple utilisant les chemins par défaut :

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    secrets:
      - source: cert
        target: /dozzle_cert.pem
      - source: key
        target: /dozzle_key.pem
    ports:
      - 7007:7007
secrets:
  cert:
    file: ./cert.pem
  key:
    file: ./key.pem
```

Ou avec des chemins personnalisés via des variables d'environnement :

```yml
services:
  agent:
    image: amir20/dozzle:latest
    command: agent
    environment:
      - DOZZLE_CERT=/certs/my-cert.pem
      - DOZZLE_KEY=/certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

Ou avec des flags en ligne de commande :

::: code-group

```sh
docker run -v /var/run/docker.sock:/var/run/docker.sock -v ./certs:/certs -p 7007:7007 amir20/dozzle:latest agent --cert /certs/my-cert.pem --key /certs/my-key.pem
```

```yaml [docker-compose.yml]
services:
  agent:
    image: amir20/dozzle:latest
    command: agent --cert /certs/my-cert.pem --key /certs/my-key.pem
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./certs:/certs
    ports:
      - 7007:7007
```

:::

> [!TIP]
> Les secrets Docker sont préférables pour fournir les certificats. Ils se créent avec la commande `docker secret create` ou, comme dans l'exemple ci-dessus, via `docker-compose.yml`. Les mêmes certificats doivent être fournis à l'instance Dozzle qui se connecte à l'agent.

Cela monte les fichiers de certificat et de clé dans l'agent. L'agent les utilise pour la communication. Les mêmes certificats doivent être fournis à l'instance Dozzle qui se connecte à l'agent.

Pour générer des certificats, vous pouvez utiliser les commandes suivantes :

```sh
$ openssl genpkey -algorithm Ed25519 -out key.pem
$ openssl req -new -key key.pem -out request.csr -subj "/C=US/ST=California/L=San Francisco/O=My Company"
$ openssl x509 -req -in request.csr -signkey key.pem -out cert.pem -days 365
```

## <Icon icon="mdi:compare-horizontal" inline /> Comparaison entre agents et connexion distante

Les agents ressemblent aux connexions distantes, mais ils ont quelques avantages. En général, les agents sont préférables aux connexions distantes pour des raisons de performance et de sécurité. Voici une comparaison :

| Fonctionnalité | Agent                         | Connexion distante                      |
| -------------- | ----------------------------- | --------------------------------------- |
| Performance    | Meilleure, charge répartie    | Moins bonne côté interface              |
| Sécurité       | SSL privé                     | Non sécurisée ou TLS Docker             |
| Simplicité     | Fonctionne d'emblée           | Nécessite d'exposer le socket Docker    |
| Permissions    | Accès complet à Docker        | Contrôlables avec un proxy              |
| Reconnexion    | Se reconnecte automatiquement | Nécessite un redémarrage de l'interface |
| Healthcheck    | Healthcheck intégré           | Pas de healthcheck                      |
| Filtres        | Prend en charge les filtres   | Pas de prise en charge des filtres      |

Si vous prévoyez d'utiliser des connexions distantes, sécurisez la connexion avec TLS Docker ou un reverse proxy.
