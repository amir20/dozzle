---
title: Moteur SQL
sourceHash: 0387115a7372
---

# Moteur SQL

Le moteur SQL est un outil puissant qui vous permet d'exécuter des requêtes SQL sur vos données. Il est conçu pour offrir une expérience fluide aux utilisateurs qui connaissent déjà SQL et souhaitent interroger leurs données dans un langage familier.

Cette fonctionnalité est actuellement en bêta et disponible pour tous les utilisateurs. Si vous avez des retours ou des suggestions, dites-le nous !

## Démarrage

Pour commencer avec le moteur SQL, il vous faut un jeu de données à interroger. Seuls les logs JSON peuvent être interrogés en SQL. Dozzle s'appuie sur WebAssembly pour exécuter les requêtes SQL dans le navigateur, ce qui signifie que vos données ne quittent jamais votre machine.

Pour commencer à utiliser le moteur SQL, assurez-vous d'avoir des logs JSON, puis ouvrez le menu déroulant et choisissez `SQL Analytics`. Il existe aussi un raccourci clavier `Ctrl+Shift+F` (ou `Cmd+Shift+F` sur macOS) pour ouvrir rapidement le moteur SQL.

## Comment ça marche ?

Le moteur SQL utilise WebAssembly pour exécuter les requêtes SQL dans le navigateur avec DuckDB. À la première ouverture du moteur SQL, DuckDB WASM est téléchargé et initialisé dans le navigateur. Cela peut prendre un moment si votre connexion est lente. Le moteur SQL lit ensuite _uniquement_ les logs JSON et crée une table virtuelle dans DuckDB. Vous pouvez ainsi exécuter des requêtes SQL sur vos données en temps réel.

La requête que Dozzle exécute au départ ressemble à ceci :

```sql
CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'
```

Cette requête crée une table nommée `logs` et déplie les logs JSON en lignes. Vous pouvez ensuite exécuter des requêtes SQL sur cette table pour analyser vos données.

## Exemples de requêtes

Voici quelques exemples de requêtes que vous pouvez exécuter avec le moteur SQL :

### Compter le nombre de logs

```sql
SELECT COUNT(*) FROM logs
```

### Filtrer les logs sur un champ précis

```sql
SELECT * FROM logs WHERE level = 'error'
```

### Regrouper les logs par un champ précis

```sql
SELECT level, COUNT(*) FROM logs GROUP BY level
```

### Interroger des champs JSON imbriqués

```sql
SELECT message.path, message.status, message.duration
FROM logs
WHERE message.status >= 400
ORDER BY message.duration DESC
```

### Agréger par fenêtre temporelle

```sql
SELECT
  date_trunc('minute', timestamp) AS minute,
  COUNT(*) AS error_count
FROM logs
WHERE level = 'error'
GROUP BY minute
ORDER BY minute DESC
```

## Limites

WebAssembly comporte quelques limites dont il faut tenir compte lors de l'utilisation du moteur SQL :

- Le moteur SQL ne prend en charge que les données structurées comme le JSON
- Le moteur SQL se limite à exécuter des requêtes dans le navigateur. Vous ne pouvez donc pas exécuter de requêtes nécessitant l'accès à des ressources ou des bases de données externes
- Le moteur SQL peut utiliser au maximum 4 Go de mémoire. Si vous arrivez à saturation, il faudra recharger la page pour libérer la mémoire
