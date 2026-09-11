---
title: Podman
sourceHash: bbfd1745527f
---

# Podman

Dozzle fonctionne avec Podman grâce à son socket compatible Docker. Une différence connue avec Docker affecte l'installation : les statistiques mémoire sont souvent absentes dans les déploiements rootless ou Quadlet (délégation des cgroups). Ce guide couvre le mode autonome (surveillance locale) et le mode agent (surveillance à distance via un serveur Dozzle central).

## Options de déploiement

| Mode         | Cas d'usage                                 | Complexité |
| ------------ | ------------------------------------------- | ---------- |
| **Autonome** | Consultation des logs d'un seul hôte        | Simple     |
| **Agent**    | Surveillance centralisée de plusieurs hôtes | Modérée    |

### Méthodes de déploiement

Podman propose plusieurs façons de lancer des conteneurs :

| Méthode           | Démarrage auto | Stats mémoire | Healthchecks | Idéal pour    |
| ----------------- | -------------- | ------------- | ------------ | ------------- |
| CLI               | Manuel         | ✓             | ✓            | Développement |
| `podman-compose`  | ✗              | ✓             | ✗            | Tests         |
| Quadlet (systemd) | ✓              | ✗\*           | ✓            | Production    |

\*Les statistiques mémoire sont généralement indisponibles en mode rootless, sauf si la délégation mémoire des cgroups v2 est activée. Voir la FAQ en bas de cette page.

---

# <Icon icon="mdi:monitor-dashboard" inline /> Mode autonome

Lancez Dozzle comme service autonome pour surveiller les conteneurs Podman locaux.

## <Icon icon="mdi:shield-account-outline" inline /> Installation rootful

Pour un démon Podman à l'échelle du système :

```bash
# Activer et démarrer le socket Podman
sudo systemctl enable podman.socket
sudo systemctl start podman.socket

# Dozzle peut se connecter via le socket Docker
podman run -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

## <Icon icon="mdi:account-outline" inline /> Installation rootless

Podman rootless isole les conteneurs dans un espace de noms utilisateur :

```bash
# Démarrer le socket utilisateur (démarre automatiquement avec la session utilisateur)
systemctl --user enable podman.socket
systemctl --user start podman.socket

# Pour un utilisateur nommé 'appuser', Dozzle peut se connecter via :
podman run -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

**Important** : un Dozzle rattaché au socket rootless d'un utilisateur ne voit que les conteneurs de cet utilisateur. Les conteneurs rootless des autres utilisateurs vivent dans des espaces de noms séparés et n'apparaîtront pas.

## <Icon icon="mdi:rocket-launch-outline" inline /> Déploiement Quadlet

Quadlet permet une gestion des conteneurs native à systemd. Créez un fichier `.container` dans `~/.config/containers/systemd/dozzle.container` :

```ini
[Unit]
Description=Dozzle Log Viewer
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

Activez et démarrez :

```bash
systemctl --user daemon-reload
systemctl --user enable --now dozzle.service
```

Sur les systèmes multi-utilisateurs, déposez le même fichier dans le `~/.config/containers/systemd/` de chaque utilisateur et choisissez un port hôte distinct par utilisateur (par ex. `PublishPort=3001:8080`). Chaque instance ne voit que les conteneurs rootless de son utilisateur.

> [!NOTE] Quadlet génère un timer systemd pour les healthchecks. `podman-compose` ne le fait pas, donc les healthchecks ne seront pas exécutés périodiquement ; déclenchez-les manuellement avec `podman healthcheck run NAME` si nécessaire.

---

# <Icon icon="mdi:lan-connect" inline /> Mode agent

Lancez Dozzle en agent sur des hôtes Podman distants pour une surveillance centralisée via un serveur Dozzle principal. Les agents communiquent avec le serveur principal en gRPC.

## <Icon icon="mdi:cog-outline" inline /> Installation de l'agent

### Prérequis

- Ouvrir le port 7007 sur l'hôte de l'agent
- Connectivité réseau entre le serveur principal et l'agent

### Démarrer l'agent Dozzle

Lancez Dozzle en mode agent sur les hôtes Podman distants :

```bash
# Agent rootful
podman run -d \
  --name dozzle-agent \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

```bash
# Agent rootless (pour l'utilisateur 'appuser')
sudo -u appuser podman run -d \
  --name dozzle-agent \
  -v /run/user/$(id -u appuser)/podman/podman.sock:/var/run/docker.sock:ro \
  -p 7007:7007 \
  ghcr.io/amir20/dozzle:latest agent
```

### Déploiement de l'agent avec Quadlet

Créez un fichier `.container` pour l'agent :

```ini
# dozzle-agent.container
[Unit]
Description=Dozzle Agent
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=7007:7007
Volume=/run/user/%U/podman/podman.sock:/var/run/docker.sock:ro
Exec=agent

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] L'entrypoint de l'image Dozzle est `/dozzle`, donc `agent` va dans `Exec=` (la commande), pas dans `Entrypoint=`.

Activez et démarrez :

```bash
systemctl --user daemon-reload
systemctl --user enable dozzle-agent.service
systemctl --user start dozzle-agent.service
```

---

# <Icon icon="mdi:server-network" inline /> Serveur principal avec agents distants

Configurez le serveur Dozzle principal pour qu'il se connecte aux agents sur les hôtes Podman distants.

## <Icon icon="mdi:cog" inline /> Configuration du serveur

Lancez le serveur Dozzle principal avec les adresses des agents :

```bash
podman run -d \
  --name dozzle \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest \
  --remote-agent "host1.example.com:7007" \
  --remote-agent "host2.example.com:7007"
```

Ou avec des variables d'environnement :

```bash
podman run -d \
  --name dozzle \
  -e DOZZLE_REMOTE_AGENT="host1.example.com:7007,host2.example.com:7007" \
  -p 3000:8080 \
  ghcr.io/amir20/dozzle:latest
```

### Serveur principal avec agents sous Quadlet

```ini
# dozzle-server.container
[Unit]
Description=Dozzle Server with Remote Agents
After=network-online.target
Wants=network-online.target

[Container]
Image=ghcr.io/amir20/dozzle:latest
PublishPort=3000:8080
Environment=DOZZLE_REMOTE_AGENT=host1.example.com:7007,host2.example.com:7007

HealthCmd=/dozzle healthcheck
HealthInterval=5s
HealthTimeout=10s
HealthRetries=5
HealthStartPeriod=15s

[Service]
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

> [!NOTE] `WantedBy=multi-user.target` ne s'applique qu'aux unités système. Pour les unités `systemctl --user`, utilisez `default.target`.

---

# <Icon icon="mdi:tune" inline /> Configuration supplémentaire

## <Icon icon="mdi:identifier" inline /> Identifiants d'hôte

Rien à configurer ici. Dozzle détermine lui-même les identifiants d'hôte sous Podman. Cette section n'existe que parce que les versions précédentes de cette page demandaient de créer un fichier qui n'a jamais rien changé.

Docker identifie un moteur par l'UUID contenu dans `/var/lib/docker/engine-id`, écrit une seule fois au premier démarrage du démon. Podman fonctionne sans démon et ne conserve aucune identité de ce type, donc son point d'accès `/info` compatible Docker remplit ce champ avec un nouvel UUID aléatoire à chaque appel. Vous pouvez le vérifier vous-même :

```sh
curl -s --unix-socket /run/user/$(id -u)/podman/podman.sock "http://d/v1.40/info" | jq .ID
curl -s --unix-socket /run/user/$(id -u)/podman/podman.sock "http://d/v1.40/info" | jq .ID
```

Deux UUID différents, et créer `/var/lib/docker/engine-id` n'y change rien, car Podman ne lit jamais ce fichier. Dozzle dérive à la place un identifiant stable à partir du nom d'hôte et du chemin de stockage des conteneurs, ce qui garde un hôte reconnaissable après un redémarrage et distingue deux utilisateurs rootless sur une même machine.

> [!WARNING] Si vous avez créé `/var/lib/docker/engine-id` sur un hôte Podman en suivant les anciennes instructions, vous pouvez le supprimer.

### En cas de collision d'identifiants

Deux hôtes Podman qui partagent à la fois le nom d'hôte et le chemin de stockage obtiennent le même identifiant dérivé, et Dozzle en écarte un comme doublon. Les noms d'hôte sont normalement distincts, il faut donc des VM clonées ou un parc où aucun nom d'hôte n'a jamais été défini. Définissez `DOZZLE_HOST_ID` sur l'un des deux pour lever l'ambiguïté :

```ini
# dozzle-agent.container
[Container]
Environment=DOZZLE_HOST_ID=web-01
```

La valeur peut contenir des lettres, des chiffres, des tirets, des underscores et des points. Elle doit être unique sur l'ensemble de vos hôtes et rester identique pendant toute la vie de l'hôte.

## <Icon icon="mdi:help-circle-outline" inline /> FAQ

### Statistiques mémoire absentes en mode rootless

Les statistiques mémoire manquent généralement dans les déploiements rootless parce que le contrôleur cgroup `memory` n'est pas délégué à la tranche utilisateur par défaut. Vérifiez ce qui est délégué :

```bash
cat /sys/fs/cgroup/user.slice/user-$(id -u).slice/cgroup.controllers
```

Si `memory` n'apparaît pas dans la sortie, activez la délégation via un fichier drop-in :

```bash
sudo mkdir -p /etc/systemd/system/user@.service.d
sudo tee /etc/systemd/system/user@.service.d/delegate.conf <<'EOF'
[Service]
Delegate=cpu cpuset io memory pids
EOF
sudo systemctl daemon-reload
```

Déconnectez-vous puis reconnectez-vous (ou redémarrez) pour que la tranche utilisateur prenne en compte la nouvelle délégation. Voir le [tutoriel Podman rootless](https://github.com/containers/podman/blob/main/docs/tutorials/rootless_tutorial.md) pour les détails.

### Healthchecks signalés comme unhealthy

**Problème avec podman-compose** : les healthchecks sont signalés comme unhealthy alors que les exécutions manuelles réussissent. C'est un comportement de Podman : les healthchecks ne sont pas évalués automatiquement sans timer systemd (Quadlet en génère un automatiquement).

Contournement avec `podman-compose` :

```bash
# Exécution manuelle du healthcheck
podman healthcheck run <container_id>
```

**Quadlet** : `HealthCmd=` attend une ligne de commande simple, pas la forme JSON `CMD [...]` de Docker :

```ini
HealthCmd=/dozzle healthcheck
```

Les anciennes versions de `podman-compose` (< 1.5.0) exécutent tous les healthchecks via `sh`, qui n'existe pas dans l'image Dozzle. Mettez à jour vers une version récente.

### Visibilité des conteneurs entre utilisateurs

Podman rootless ne peut accéder qu'aux conteneurs du même espace de noms utilisateur. Si Dozzle tourne sous un utilisateur, il ne peut pas voir les conteneurs de la session rootless d'un autre utilisateur.

**Solution** : lancez Dozzle sous le même utilisateur, ou utilisez le mode rootful.
