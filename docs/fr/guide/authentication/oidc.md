---
title: OpenID Connect
sourceHash: 9dcc3df4a715
---

# <Icon icon="mdi:shield-account" inline /> OpenID Connect

Avec `--auth-provider oidc`, votre fournisseur d'identité est la base des utilisateurs. Dozzle ne lit jamais `users.yml` dans ce mode : qui est un utilisateur, quels rôles il possède et quels conteneurs il peut voir viennent tous du token OpenID Connect. Ajoutez un utilisateur dans Keycloak, Authentik, Zitadel ou Pocket ID et il peut se connecter ; retirez-lui son rôle et il ne le peut plus.

C'est différent de [connecter les utilisateurs de `users.yml` avec OIDC](/fr/guide/authentication/oauth#se-connecter-avec-oidc) sous le fournisseur `simple`. Là, le fournisseur prouve seulement qui vous êtes et c'est toujours `users.yml` qui décide de ce que vous obtenez. Choisissez `simple` quand vous voulez lister chaque utilisateur à la main, et `oidc` quand c'est le fournisseur qui doit posséder la liste.

## Configuration minimale

Enregistrez Dozzle comme client confidentiel auprès de votre fournisseur et réglez l'URI de redirection sur :

```
https://your-dozzle-host/api/auth/callback
```

Incluez le chemin de base si Dozzle tourne sous un chemin de base, par exemple `https://example.com/dozzle/api/auth/callback`. Pointez ensuite Dozzle sur l'émetteur :

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider oidc --auth-oidc-issuer https://keycloak.example.com/realms/main --auth-oidc-client-id dozzle --auth-oidc-client-secret secret
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: oidc
      DOZZLE_AUTH_OIDC_ISSUER: https://keycloak.example.com/realms/main
      DOZZLE_AUTH_OIDC_CLIENT_ID: dozzle
      DOZZLE_AUTH_OIDC_CLIENT_SECRET: secret
```

:::

C'est tout. Aucun chemin de claim n'a besoin d'être défini pour les dispositions courantes, et `DOZZLE_AUTH_OIDC_NAME` ne change que le libellé du bouton de connexion. Le client secret accepte aussi un équivalent `_FILE`, voir [Utiliser les secrets Docker](/fr/guide/authentication/oauth#utiliser-les-secrets-docker-pour-le-client-secret).

L'URL de l'émetteur est celle qui sert `/.well-known/openid-configuration`. Dozzle récupère ce document pour trouver les endpoints d'autorisation, de token et de userinfo, et refuse de démarrer le flux si le document déclare un émetteur différent de celui que vous avez configuré.

## Rôles

Les rôles d'un utilisateur sont lus depuis le token. Dozzle essaie ces claims dans l'ordre et prend le premier qui existe :

1. `dozzle_roles`
2. `resource_access.<client-id>.roles`
3. `roles`

Le client id est déjà configuré, donc avec `DOZZLE_AUTH_OIDC_CLIENT_ID=dozzle` le deuxième chemin est `resource_access.dozzle.roles`, qui est l'endroit où Keycloak place les rôles de client. Rien d'autre n'a besoin d'être défini pour cette disposition.

Si vos rôles se trouvent ailleurs, `DOZZLE_AUTH_OIDC_ROLES_CLAIM` remplace la recherche par le seul chemin, séparé par des points, que vous lui donnez :

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: realm_access.roles
```

Chaque claim est cherché d'abord dans l'ID token puis dans la réponse userinfo, peu importe donc dans lequel des deux votre fournisseur le place. Trois formes sont acceptées : un tableau de chaînes, une seule chaîne séparée par des virgules ou des espaces, et un objet dont les clés sont les rôles, ce qui est la façon dont Zitadel encode les rôles de projet.

Les noms de rôles sont les mêmes que dans `users.yml` : `shell`, `actions`, `download`, `notifications`, `cloud` et `all`, avec `^` pour soustraire, donc `all,^shell` accorde tout sauf l'accès au shell. Les noms préfixés par `dozzle_` sont acceptés aussi, ce qui aide quand le fournisseur partage un même claim de rôles entre plusieurs applications. Voir [rôles](/fr/guide/authentication/simple#definir-des-roles-specifiques-pour-les-utilisateurs) pour ce que chacun débloque.

> [!WARNING]
> `groups` est volontairement absent de la liste. Chez Authentik ou Google, chaque utilisateur appartient à au moins un groupe, donc le chercher transformerait « refusé » en « connecté et peut lire tous les conteneurs ». Si ce sont des groupes que vous avez, transposez-les en un claim `dozzle_roles` côté fournisseur, voir les exemples ci-dessous.

### Quand une connexion est refusée

La connexion est rejetée quand aucun des claims n'existe, ou quand le premier qui existe est vide. Le log nomme les chemins qui ont été essayés :

```
WRN OIDC login rejected: no roles claim found in the ID token or userinfo, or it was empty sub=... tried="dozzle_roles, resource_access.dozzle.roles, roles"
```

Un claim présent mais qui ne contient rien que Dozzle reconnaisse comme un rôle est un cas différent. Cet utilisateur se connecte sans aucun privilège, comme avec `roles: none` dans `users.yml` : il peut lire les logs des conteneurs que son filtre autorise, et rien de plus. Les rôles de realm de Keycloak se comportent ainsi, parce que chaque utilisateur y porte `offline_access` et `uma_authorization`, c'est pourquoi les exemples ci-dessous utilisent des rôles de client à la place.

## Filtres

Les filtres de conteneurs fonctionnent de la même façon, lus depuis le premier de `dozzle_filters`, `resource_access.<client-id>.filters` et `filters` qui existe, ou depuis le seul chemin donné dans `DOZZLE_AUTH_OIDC_FILTERS_CLAIM`. Chaque valeur est un filtre dans la [même syntaxe que `users.yml`](/fr/guide/authentication/simple#definir-des-filtres-specifiques-pour-les-utilisateurs), par exemple `label=com.example.app` ou `name=web` :

```json
"resource_access": {
  "dozzle": {
    "roles": ["shell", "actions"],
    "filters": ["label=com.example.app"]
  }
}
```

Un utilisateur sans claim de filtres voit tous les conteneurs que l'instance Dozzle peut voir. Un filtre qui ne se parse pas fait échouer la connexion au lieu d'être ignoré, une faute de frappe côté fournisseur ne peut donc pas élargir discrètement ce que quelqu'un voit.

## Identité

Le claim `sub` est l'identifiant stable de l'utilisateur. Il sert de clé au répertoire de profil sous `/data`, les réglages suivent donc la personne même si son nom d'utilisateur ou son email change chez le fournisseur. Le nom affiché dans le menu est `name`, avec repli sur `preferred_username`, puis `email`, puis `sub`. `email` et `picture` alimentent l'avatar, et une URL `picture` est utilisée directement quand le fournisseur en envoie une.

Contrairement au fournisseur `simple`, l'email n'a pas besoin d'être vérifié ici. Il est seulement affiché, jamais comparé à quoi que ce soit, un émetteur qui n'accorde aucun scope email fonctionne donc très bien.

## Sessions

Après la connexion, Dozzle émet son propre cookie de session portant les rôles et les filtres qu'il a lus dans le token. Ils sont appliqués à chaque requête, mais ne sont pas récupérés à nouveau : un changement de rôle chez le fournisseur prend effet à la prochaine connexion de l'utilisateur. Si cet écart compte, réglez [`--auth-ttl`](/fr/guide/supported-env-vars) sur quelque chose comme `8h` pour que les sessions expirent et soient rétablies à partir d'un token frais.

## Déconnexion

Se déconnecter efface la session de Dozzle. Si `--auth-logout-url` est défini, le navigateur y est ensuite envoyé, pointez-le donc sur l'URL de fin de session de votre fournisseur pour déconnecter aussi l'utilisateur du fournisseur :

```yaml
DOZZLE_AUTH_LOGOUT_URL: https://keycloak.example.com/realms/main/protocol/openid-connect/logout
```

## Ce qui diffère de `simple`

Les deux fournisseurs partagent les flags `--auth-oidc-*`, la différence se voit donc dans le comportement :

- `users.yml` n'est jamais lu. S'il en existe un sous `/data`, Dozzle journalise qu'il l'ignore.
- Il n'y a ni formulaire de mot de passe ni endpoint `/api/token`. Le fournisseur d'identité est le seul moyen d'entrer, donc une URL de callback erronée ou un client secret expiré verrouille tout le monde jusqu'à ce que ce soit réparé.
- `--auth-github-*` est une erreur au démarrage. GitHub n'est pas un émetteur OpenID Connect et ne publie aucun claim d'où lire des rôles.
- Le démarrage journalise l'émetteur et les chemins de claims depuis lesquels les rôles seront lus.

## Exemples de fournisseurs

### Keycloak

Créez un client `dozzle` dans votre realm avec l'authentification du client activée, et ajoutez l'URI de redirection ci-dessus. Puis, sous l'onglet **Roles** du client, créez les rôles de client que vous voulez distribuer : `shell`, `actions`, `download`, `notifications`, `cloud` ou `all`. Attribuez-les aux utilisateurs ou aux groupes sous **Role mapping**.

Keycloak émet les rôles de client sous `resource_access.<client-id>.roles`, que Dozzle cherche déjà. Vérifiez les **Client scopes** du client, ouvrez le scope dédié et confirmez que le mapper **client roles** ajoute le claim à l'ID token ou au userinfo ; Dozzle lit les deux mais pas l'access token.

Pour les filtres, ajoutez un attribut utilisateur `dozzle_filters` et un mapper **User Attribute** sur le scope dédié avec le même nom de claim dans le token et **Multivalued** activé. Chaque valeur est un filtre, par exemple `label=com.example.app`.

### Authentik

Ajoutez un scope mapping sous **Customization** → **Property Mappings** qui renvoie les rôles à partir des groupes de l'utilisateur, et attachez-le au fournisseur Dozzle :

```python
roles = []
if request.user.ak_groups.filter(name="dozzle-admins").exists():
    roles.append("all")
elif request.user.ak_groups.filter(name="dozzle-users").exists():
    roles.append("download")
return {"dozzle_roles": roles}
```

Un utilisateur qui n'est dans aucun des deux groupes obtient une liste vide et est refusé.

### Zitadel

Accordez aux utilisateurs des rôles de projet nommés d'après les rôles Dozzle et activez **Assert Roles on Authentication** sur l'application. Le claim de rôles de Zitadel est un objet dont les clés sont les noms de rôles, ce que Dozzle accepte, réglez donc :

```yaml
DOZZLE_AUTH_OIDC_ROLES_CLAIM: urn:zitadel:iam:org:project:roles
```

### Google

Les tokens de Google ne portent aucun claim de rôles, `oidc` ne peut donc pas être utilisé avec lui. Utilisez plutôt le [fournisseur `simple` avec la connexion Google](/fr/guide/authentication/oauth#google), où c'est `users.yml` qui décide qui entre.
