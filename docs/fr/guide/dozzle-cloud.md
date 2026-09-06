---
title: Dozzle Cloud
sourceHash: 34c0056128a5
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) est un compagnon géré et facultatif de Dozzle auto-hébergé. Il relie vos instances entre elles, résume les événements des conteneurs, distribue les alertes sur plusieurs canaux et vous permet de poser des questions sur votre infrastructure depuis une messagerie. Dozzle reste entièrement open source et auto-hébergé ; Cloud vient se poser par-dessus.

L'objectif est que Dozzle Cloud ressemble à l'assistant SRE personnel dont vous ignoriez avoir besoin : il surveille vos conteneurs, vous prévient quand quelque chose compte, et se fait oublier le reste du temps.

## Fonctionnalités

### <Icon icon="mdi:text-box-outline" inline /> Résumés de logs

Les événements des conteneurs sont regroupés et résumés par un LLM. Chaque résumé indique la gravité, le conteneur source et un lien vers la ligne de log complète dans votre instance Dozzle.

### <Icon icon="mdi:group" inline /> Regroupement de motifs

Les erreurs répétées sont regroupées et comptées au lieu d'être envoyées une par une. Une boucle qui émet 200 fois la même exception produit une seule notification avec une fréquence, pas 200.

### <Icon icon="mdi:robot-outline" inline /> Agent IA

Un agent conversationnel répond aux questions sur l'état des conteneurs et l'activité récente des logs. Il est disponible sur Telegram et Discord.

Avec les formules Pro et Team, l'agent peut aussi agir sur les conteneurs (démarrer, arrêter, redémarrer) directement depuis la conversation, sans nécessiter d'accès shell à l'hôte.

### <Icon icon="mdi:calendar-clock" inline /> Récapitulatifs quotidiens

Un résumé planifié de l'activité récente sur vos instances liées : principaux motifs d'erreur, nombre d'événements et santé globale. Envoyé par e-mail à l'heure et dans le fuseau horaire que vous configurez.

### <Icon icon="mdi:bell-ring-outline" inline /> Canaux de notification

Les alertes peuvent être routées vers plusieurs canaux en parallèle. Chaque canal peut être activé ou désactivé indépendamment et limité à certaines instances Dozzle.

| Canal                                                         | Alertes | Récapitulatif quotidien | Agent bidirectionnel |
| ------------------------------------------------------------- | :-----: | :---------------------: | :------------------: |
| <Icon icon="mdi:telegram" inline /> Telegram                  |    ✓    |            ✓            |          ✓           |
| <Icon icon="ic:baseline-discord" inline /> Discord            |    ✓    |            ✓            |          ✓           |
| <Icon icon="mdi:email-outline" inline /> E-mail               |    ✓    |            ✓            |                      |
| <Icon icon="mdi:slack" inline /> Slack                        |    ✓    |                         |                      |
| <Icon icon="simple-icons:ntfy" inline /> ntfy                 |    ✓    |                         |                      |
| <Icon icon="mdi:webhook" inline /> Webhooks                   |    ✓    |                         |                      |
| <Icon icon="mdi:bell-badge-outline" inline /> Push navigateur |    ✓    |                         |                      |

### <Icon icon="mdi:bell-sleep-outline" inline /> Mise en sourdine des notifications

Les notifications peuvent être coupées pendant une heure, huit heures, jusqu'au lendemain matin ou jusqu'à la semaine suivante. Pratique pendant un incident ou une maintenance planifiée.

### <Icon icon="mdi:view-dashboard-outline" inline /> Tableau de bord multi-instances

Les instances Dozzle liées apparaissent dans un tableau de bord unique. Chaque instance s'authentifie avec une clé d'API, sans agent supplémentaire sur l'hôte. Le tableau de bord affiche l'état de connexion, l'inventaire des conteneurs et le flux de logs en direct.

### <Icon icon="mdi:database-search-outline" inline /> Recherche plein texte dans les logs

Chaque ligne de log transmise par vos instances liées est écrite dans un index de recherche plein texte. Vous pouvez interroger toutes les instances d'un coup, ou filtrer par conteneur, gravité ou plage de temps. Les recherches renvoient des résultats en quelques millisecondes, même sur des semaines d'historique, et chaque correspondance renvoie au contexte environnant dans l'instance source. La rétention dépend de la formule et va de 24 heures à 30 jours.

### <Icon icon="mdi:shield-lock-outline" inline /> Sécurité

- Les clés d'API sont hachées avec BLAKE2b et peuvent expirer.
- La connexion se fait via OAuth GitHub ou Google.
- Les logs et le contenu des événements ne sont conservés que pendant la fenêtre de rétention de votre formule.

## Lier une instance

Pour relier un Dozzle auto-hébergé à Dozzle Cloud :

1. Ouvrez votre instance Dozzle et cliquez sur l'icône **cloud** dans la barre supérieure.
2. Cliquez sur **Link instance**. Vous serez redirigé pour vous authentifier et confirmer la connexion.
3. Une fois lié, configurez les abonnements aux alertes dans Dozzle pour choisir quels événements sont transmis.

## Contrôler ce qui est transmis

Par défaut, chaque conteneur en cours d'exécution envoie ses logs à Dozzle Cloud tant que l'instance est liée. Pour les conteneurs bavards dont le bruit en niveau info n'a aucune valeur de diagnostic, vous pouvez filtrer ou désactiver complètement l'envoi par conteneur avec un seul label.

### `dev.dozzle.cloud.min_level`

| Valeur                                        | Effet                                                                                                         |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| _(non défini)_                                | Toutes les lignes de log sont transmises. Comportement par défaut.                                            |
| `disabled`                                    | Le conteneur est complètement ignoré. Aucun log n'est transmis à Cloud.                                       |
| `trace`                                       | Identique à non défini, puisque trace est le niveau le plus bas. Tout est transmis.                           |
| `debug` / `info` / `warn` / `error` / `fatal` | Seules les lignes de ce niveau ou supérieur sont transmises. Les lignes sans niveau détecté passent toujours. |

Une valeur non reconnue (une faute de frappe comme `warning` ou `wran`) est signalée comme une erreur puis ignorée, et le conteneur transmet donc tout comme si le label n'existait pas.

Le label est lu au démarrage du lecteur de logs. Le modifier sur un conteneur en cours d'exécution ne prend effet qu'après son redémarrage.

```yaml
services:
  zigbee2mqtt:
    image: koenkk/zigbee2mqtt
    labels:
      # Ne transmettre que warn/error/fatal à Dozzle Cloud
      - dev.dozzle.cloud.min_level=warn

  noisy-debug-tool:
    image: example/debug
    labels:
      # Ne rien envoyer depuis ce conteneur
      - dev.dozzle.cloud.min_level=disabled
```

Le filtre s'applique sur votre instance Dozzle avant que les logs ne quittent l'hôte, donc les lignes écartées ne passent jamais par le réseau et ne comptent pas dans votre formule. La consultation locale des logs dans Dozzle n'est pas affectée.

## Tarifs

La formule gratuite est volontairement généreuse : vous devriez pouvoir vous servir réellement de Dozzle Cloud sur un homelab ou dans une petite équipe sans buter sur un mur. Des formules payantes existent pour des volumes d'événements plus élevés, une rétention plus longue et les actions de l'agent sur les conteneurs. Voir [cloud.dozzle.dev](https://cloud.dozzle.dev) pour les limites et les détails des formules à jour.

## Retours

Dozzle Cloud est développé par la personne qui a créé Dozzle, avec la même exigence : des choses que les gens ont vraiment envie d'utiliser. Si vous l'essayez et que quelque chose vous semble bancal, manquant ou vraiment utile, [ouvrez une discussion](https://github.com/amir20/dozzle/discussions). Ces retours orientent ce qui sera construit ensuite.
