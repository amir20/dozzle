---
title: Qu'est-ce que Dozzle ?
sourceHash: a37de5cd15f3
---

# Qu'est-ce que Dozzle ?

Dozzle est un projet open source sponsorisé par Docker OSS. C'est un visualiseur de logs léger, accessible depuis le navigateur, conçu pour simplifier la surveillance et le débogage des applications conteneurisées sur Docker, Docker Swarm et Kubernetes.

Vous découvrez le projet ? Allez directement à [Démarrage](/fr/guide/getting-started) pour le lancer en moins d'une minute.

## Fonctionnalités principales

### <Icon icon="mdi:pulse" inline /> Surveillance en temps réel

Diffusez les logs des conteneurs en cours d'exécution avec une mise à jour instantanée. Métriques CPU, mémoire et réseau en direct, avec visualisation de l'historique.

### <Icon icon="mdi:cube-outline" inline /> Déploiement flexible

Fonctionne en [serveur autonome](/fr/guide/getting-started), en déploiement [Swarm](/fr/guide/swarm-mode), en installation [Kubernetes](/fr/guide/k8s), ou avec des [agents distants](/fr/guide/agent) répartis sur plusieurs hôtes.

### <Icon icon="mdi:text-search" inline /> Traitement avancé des logs

Détection automatique du JSON et coloration, regroupement des traces d'appels multilignes, [filtres](/fr/guide/filters) et un [moteur SQL](/fr/guide/sql-engine) intégré pour les requêtes ponctuelles.

### <Icon icon="mdi:server-network" inline /> Support multi-hôtes

Surveillez les conteneurs de plusieurs hôtes Docker depuis une seule interface. Voir les [agents](/fr/guide/agent).

### <Icon icon="mdi:console" inline /> Terminal interactif

Attachez-vous ou exécutez des commandes dans les conteneurs depuis le navigateur. Voir [Accès shell](/fr/guide/shell).

### <Icon icon="mdi:gesture-tap-button" inline /> Actions sur les conteneurs

Démarrez, arrêtez, redémarrez et mettez à jour les conteneurs directement depuis l'interface. Voir [Actions](/fr/guide/actions).

### <Icon icon="mdi:bell-ring-outline" inline /> Alertes et webhooks

Définissez des motifs de log qui déclenchent des notifications vers Slack, Discord, e-mail et bien d'autres. Voir [Alertes et webhooks](/fr/guide/alerts-and-webhooks).

### <Icon icon="mdi:shield-lock-outline" inline /> Authentification

Laissez l'accès ouvert, ou ajoutez une [authentification simple ou par proxy](/fr/guide/authentication) avec contrôle d'accès par rôle.

### <Icon icon="mdi:feather" inline /> Léger et rapide

Backend en Go, frontend en Vue 3, diffusion via SSE et WebSocket, pour une empreinte minimale.

## Étapes suivantes

- [Démarrage](/fr/guide/getting-started)
- [Variables d'environnement supportées](/fr/guide/supported-env-vars)
- [FAQ](/fr/guide/faq)

Dozzle est sous licence MIT et activement maintenu.
