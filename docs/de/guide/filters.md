---
title: Filter
sourceHash: e380a612fd7f
---

# Container filtern

<Badge type="tip" text="Docker" />
<Badge type="tip" text="K8s" />

Dozzle unterstützt mit `DOZZLE_FILTER` oder `--filter` ein bedingtes Filtern ähnlich zu Dockers [--filter](https://docs.docker.com/reference/cli/docker/container/ls/#filter). Die Filter werden direkt an Docker weitergereicht und schränken ein, was Dozzle überhaupt sieht. Nach einem Label filterst du zum Beispiel mit `--filter "label=color"`, analog zum Befehl `docker ps --filter "label=color"`.

::: code-group

```sh
docker run --volume=/var/run/docker.sock:/var/run/docker.sock -p 8080:8080 amir20/dozzle --filter label=color
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - 8080:8080
    environment:
      DOZZLE_FILTER: label=color
```

:::

Gängige Filter sind `name` oder `label`, um Dozzles Zugriff auf Container einzugrenzen.

## Filter für Oberfläche, Agents und Benutzer

Dozzle unterstützt mehrere Filter, um die sichtbaren Container einzuschränken. Filter lassen sich auf Ebene der Oberfläche, des Agents oder des Benutzers setzen.

1. **Filter der Oberfläche**: Diese Filter gelten für die Dozzle-Instanz und werden an Docker geschickt, um die sichtbaren Container einzuschränken. Sie wirken auf alle Agents und alle Benutzer, die keine eigenen Filter haben.
2. **Agent-Filter**: Diese Filter werden auf Agent-Ebene gesetzt und an Docker geschickt, um die von diesem Agent bereitgestellten Container einzuschränken. Agent-Filter und Filter der Oberfläche wirken zusammen.
3. **Benutzerfilter**: Diese Filter werden pro Benutzer gesetzt und bestimmen, welche Container dieser Benutzer sieht. Sind keine Benutzerfilter definiert, verwendet Dozzle die Filter der Oberfläche.

Mehr zum Setzen von Filtern für einzelne Benutzer findest du unter [Benutzerfilter](/de/guide/authentication#setting-specific-filters-for-users). Details zu Filtern für Agents stehen unter [Agent-Filter](/de/guide/agent#setting-up-filters).

> [!WARNING]
> Wichtig zu verstehen: Mehrere Filter werden kombiniert, um die Container einzuschränken. Setzt du zum Beispiel `--filter label=color` auf Ebene der Oberfläche und `--filter label=type` auf Agent-Ebene, zeigt Dozzle nur Container an, die sowohl das Label `color` als auch `type` haben.
