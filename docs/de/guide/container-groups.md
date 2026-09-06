---
title: Container-Gruppen
sourceHash: 87c26dbd0b16
---

# Container-Gruppen

Dozzle gruppiert Container automatisch anhand ihres Stack- oder Service-Namens. Zusätzlich kannst du mit Labels eigene Gruppen anlegen.

## Standardgruppen

Im Host-Modus werden Container standardmäßig nach ihrem Stack-Namen gruppiert. Ist das Label `com.docker.swarm.service.name` vorhanden, aktiviert Dozzle automatisch einen "Swarm-Modus", in dem alle Container mit demselben Service-Namen zusammengefasst werden.

## Eigene Gruppen

Darüber hinaus kannst du eigene Gruppen erstellen, indem du deinem Container ein Label hinzufügst. Das Label heißt `dev.dozzle.group`, der Wert ist der Name der Gruppe. Alle Container mit demselben Gruppennamen werden in der Oberfläche zusammengefasst. Hast du zum Beispiel eine Gruppe `myapp`, werden alle Container mit dem Label `dev.dozzle.group=myapp` zusammengefasst.

Hier ein Beispiel mit Docker Compose oder der Docker CLI:

::: code-group

```sh
docker run --label dev.dozzle.group=myapp hello-world
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: hello-world
    labels:
      - dev.dozzle.group=myapp
```

:::
