---
title: Relier votre instance
sourceHash: ffd948b27c63
---

# Relier votre instance

Relier un Dozzle auto-hébergé à [Dozzle Cloud](/fr/guide/dozzle-cloud), vérifier qu'il est réellement connecté, et corriger quand il ne l'est pas.

## Relier une instance

1. Ouvrez votre Dozzle auto-hébergé et cliquez sur l'icône **nuage** dans la barre du haut.
2. Cliquez sur **Link instance**. Vous êtes envoyé vers Cloud pour vous connecter avec GitHub ou Google et confirmer.
3. L'instance apparaît sur le tableau de bord Cloud en quelques secondes.

Aucun mot de passe à créer, aucun agent à installer sur l'hôte.

## Vous n'avez besoin ni d'IP publique, ni de port ouvert, ni de domaine

C'est l'inquiétude la plus fréquente, et la réponse est non sur les trois points.

Votre instance Dozzle ouvre une connexion **sortante** vers Cloud et la maintient ouverte. Cloud ne se reconnecte jamais vers vous, ne scanne jamais votre hôte et n'a jamais besoin d'atteindre votre adresse. Cela fonctionne donc normalement quand Dozzle est :

- derrière du NAT sur un réseau domestique, sans redirection de port
- sur une adresse privée RFC1918 comme `192.168.1.50`
- sur un réseau Tailscale, WireGuard ou ZeroTier
- derrière du CGNAT, où vous ne pourriez pas rediriger un port même en le voulant
- sur un portable qui change de réseau

Ni reverse proxy, ni nom DNS dynamique, ni IP fixe ne sont nécessaires.

## Règles de pare-feu

Seul l'accès **sortant** est requis. Autorisez votre hôte Dozzle à joindre :

```
agent.doligence.dozzle.dev:443    (TCP, outbound)
```

Cette seule destination sur le port 443 suffit. Si votre pare-feu filtre par nom d'hôte plutôt que par IP, autorisez le nom : les adresses derrière peuvent changer. La plupart des pare-feux domestiques et de petite entreprise autorisent déjà tout le trafic sortant, donc en général il n'y a rien à configurer.

## Où vivent les règles et les canaux

Presque tout le monde bute là-dessus, donc autant le dire clairement.

| Ce que vous voulez changer                                      | Où le faire                                                    |
| --------------------------------------------------------------- | -------------------------------------------------------------- |
| **Ce qui déclenche une alerte** — conteneurs, motifs, seuils    | Dozzle auto-hébergé → [Alertes](/fr/guide/alerts-and-webhooks) |
| **Où les alertes sont livrées** — e-mail, Telegram, Slack, ...  | Dozzle Cloud → [Canaux](/fr/guide/dozzle-cloud/channels)       |
| Revoir les alertes passées, mettre en sourdine, changer d'offre | Dozzle Cloud                                                   |

La règle est définie sur votre propre instance parce que c'est là que sont vos logs. La distribution est configurée dans Cloud parce que c'est ce qui tient la connexion vers votre téléphone. Si vous cherchez dans Cloud un endroit pour dire « préviens-moi quand ce conteneur part en erreur » sans le trouver, c'est pour ça : ouvrez plutôt votre Dozzle auto-hébergé.

## Relier plus d'une instance

Chaque instance se relie séparément, avec les mêmes étapes et le même compte Cloud. Une fois reliées, elles apparaissent toutes ensemble sur le tableau de bord, et les questions posées dans le chat couvrent toutes les instances connectées d'un coup.

Cette vue combinée vit dans Cloud, pas dans un Dozzle auto-hébergé en particulier. Un Dozzle auto-hébergé affiche les hôtes que vous y avez configurés directement ; il n'affiche pas les autres instances reliées.

Relier plusieurs instances Dozzle est différent des fonctionnalités propres à Dozzle que sont le [Mode agent](/fr/guide/agent) et les [Hôtes distants](/fr/guide/remote-hosts), qui rattachent des hôtes Docker supplémentaires à un seul Dozzle. Les deux sont pris en charge et peuvent se combiner.

L'offre gratuite relie une instance à la fois. Voir [Offres et limites](/fr/guide/dozzle-cloud/plans).

## Rien n'apparaît

Reprenez ces points dans l'ordre.

**1. Le conteneur Dozzle tourne-t-il ?**
Si Dozzle lui-même est arrêté ou en redémarrage, rien n'atteint Cloud.

**2. Le lien a-t-il été mené à son terme ?**
Commencer le lien sans l'approuver ne laisse rien. Refaites les étapes ci-dessus et vérifiez que l'instance apparaît ensuite sur la page Instances.

**3. La clé d'API a-t-elle été supprimée ?**
Supprimer la clé d'API d'une instance la dissocie définitivement. L'ancienne clé ne peut pas être réattachée : reliez à nouveau pour en obtenir une nouvelle.

**4. Le trafic sortant est-il bloqué ?**
Les réseaux restrictifs (entreprise, université, certains hébergeurs VPS) peuvent bloquer le 443 sortant vers des destinations absentes d'une liste d'autorisation. Voir _Règles de pare-feu_.

**5. Avez-vous atteint la limite d'instances de l'offre gratuite ?**
L'offre gratuite relie une instance à la fois. Tenter d'en relier une deuxième affiche un message de limite au lieu de connecter.

**6. Le conteneur est-il exclu de la transmission ?**
Un conteneur portant le label `dev.dozzle.cloud.min_level=disabled` n'envoie rien, par conception. Si un conteneur précis manque alors que les autres fonctionnent, vérifiez ses labels. Voir [Vos données](/fr/guide/dozzle-cloud/your-data).

## Laisser l'agent piloter les conteneurs

Lire les logs et l'état des conteneurs fonctionne dès qu'une instance est reliée. Démarrer, arrêter et redémarrer sont refusés par votre instance tant que vous ne les activez pas vous-même :

::: code-group

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle
    environment:
      DOZZLE_ENABLE_ACTIONS: true
```

```sh
docker run ... amir20/dozzle --enable-actions
```

:::

C'est un réglage sur **votre** Dozzle, pas dans Cloud, car il gouverne ce que votre Dozzle accepte de faire à vos conteneurs. Redémarrez Dozzle après l'avoir changé. Voir [Actions](/fr/guide/actions).

Une fois relié, voir [Dans votre Dozzle](/fr/guide/dozzle-cloud/in-dozzle) pour ce qui apparaît dans votre propre interface.

## Dissocier

Supprimez la clé d'API de l'instance sur la page Instances dans Cloud. La connexion tombe, plus aucune donnée n'est transmise, et votre Dozzle auto-hébergé continue de fonctionner exactement comme avant. Relier ne change jamais la consultation locale des logs.
