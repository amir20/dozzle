---
title: Einrichtungsassistent
sourceHash: 413c966adbb1
---

# Einrichtungsassistent

<Badge type="warning" text="Docker Only" />

Eine frische Dozzle-Installation startet mit einem kurzen Einrichtungsassistenten. Er führt dich durch die wenigen Dinge, die die meisten direkt nach der Installation ändern: Login einschalten, Container-Aktionen und Shell-Zugriff erlauben und Dozzle Cloud verbinden. Alles, was er speichert, lässt sich auch über Flags oder Umgebungsvariablen setzen, der Assistent ist also optional.

Der Assistent erscheint nur bei einer frischen Installation im Server-Modus. Swarm- und Kubernetes-Deployments zeigen ihn nie. Du kannst ihn später in den Einstellungen erneut öffnen.

## <Icon icon="mdi:format-list-numbered" inline /> Schritte

### 1. Login

Der Login kommt zuerst, damit auf einer Instanz, die jeder erreichen kann, sonst nichts geändert werden kann.

Zuerst prüft der Assistent, ob `/data` auf einem Volume liegt. Einstellungen und Benutzer werden dort gespeichert, und ohne Volume wären sie beim nächsten Neuerstellen des Containers weg. Ist `/data` nicht persistent, zeigt der Assistent, wie du es einbindest, und wartet, bis du auf **Erneut prüfen** klickst.

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle_data:/data
    ports:
      - 8080:8080
volumes:
  dozzle_data:
```

Sobald `/data` persistent ist, wählst du eine von drei Optionen:

- **Dozzle-Konto** legt einen einzelnen Benutzer mit Benutzername, optionaler E-Mail und Passwort an. Dozzle schreibt `/data/users.yml` und setzt `authProvider: simple`. Unter [Einfache Authentifizierung](/de/guide/authentication/simple) erfährst du, wie du später weitere Benutzer oder Rollen hinzufügst.
- **Mein Proxy** ist für Authelia, Authentik, Cloudflare Access und Ähnliches. Dozzle vertraut dem Header `Remote-User`, veröffentliche also nur den Proxy und nie den Port von Dozzle selbst. Das setzt `authProvider: forward-proxy`. Siehe [Forward Proxy](/de/guide/authentication/forward-proxy).
- **OIDC** zeigt einen Link zur Anleitung für [OpenID Connect](/de/guide/authentication/oidc) und die Umgebungsvariablen, die du ergänzen musst. OIDC braucht ein Client-Secret, deshalb wird hier nichts geschrieben und du richtest es selbst ein.

Ist Dozzle nur in deinem eigenen Netzwerk erreichbar, überspringt **Ohne Anmeldung fortfahren** diesen Schritt.

Nachdem ein Konto oder Proxy gespeichert wurde, startet Dozzle sofort neu, damit der Login aktiv ist, bevor irgendetwas anderes geändert wird. Du landest auf der Login-Seite, und nach der Anmeldung macht der Assistent mit dem nächsten Schritt weiter.

### 2. Aktionen und Shell

Zwei Schalter legen fest, was Dozzle mit deinen Containern tun darf:

- **Starten, Stoppen und Neustarten** schaltet [Container-Aktionen](/de/guide/actions) ein (`enableActions`).
- **Shell** schaltet das [Anhängen und Ausführen von Befehlen](/de/guide/shell) in Containern ein (`enableShell`). Standardmäßig ist es aus. Shell-Zugriff auf einen Container ist oft so viel wert wie Zugriff auf den Host, schalte ihn also nur ein, wenn du ihn brauchst.

Ist eine Einstellung bereits über ein Flag oder eine Umgebungsvariable festgelegt, ist ihr Schalter schreibgeschützt und weist darauf hin.

### 3. Dozzle Cloud

[Dozzle Cloud](/de/guide/dozzle-cloud) schickt Alerts, sobald etwas kaputtgeht, eine morgendliche Zusammenfassung dessen, was zu beheben ist, und bewahrt einen Verlauf, der Neustarts übersteht. **Dozzle Cloud verbinden** verknüpft diese Instanz, **Nicht jetzt** geht weiter. Dieser Schritt entfällt, wenn die Instanz bereits verknüpft ist oder du sie nicht verknüpfen darfst.

### 4. Neustart

Der letzte Schritt listet die Änderungen auf, die gespeichert, aber noch nicht aktiv sind. **Dozzle neu starten** startet den Container neu, wartet, bis er wieder da ist, und lädt die Seite neu. Steht nichts aus, meldet der Schritt nur, dass du fertig bist.

Kann sich Dozzle nicht selbst neu starten (zum Beispiel, wenn es seinen eigenen Container nicht findet), zeigt der Assistent stattdessen die Umgebungsvariablen, die du in deine Compose-Datei eintragen kannst.

## <Icon icon="mdi:file-cog-outline" inline /> Wo Einstellungen gespeichert werden

Der Assistent speichert deine Auswahl in `/data/dozzle.yml`. Dozzle liest diese Datei einmal beim Start, deshalb brauchen Änderungen einen Neustart. Dozzle startet sich aus dem Assistenten heraus selbst neu, du musst das also nicht von Hand erledigen.

```yaml [/data/dozzle.yml]
authProvider: simple
enableActions: true
enableShell: false
```

| Schlüssel       | Werte                             | Entspricht              |
| --------------- | --------------------------------- | ----------------------- |
| `authProvider`  | `none`, `simple`, `forward-proxy` | `DOZZLE_AUTH_PROVIDER`  |
| `enableActions` | `true`, `false`                   | `DOZZLE_ENABLE_ACTIONS` |
| `enableShell`   | `true`, `false`                   | `DOZZLE_ENABLE_SHELL`   |

Flags und Umgebungsvariablen haben immer Vorrang vor der Datei. Ist `DOZZLE_ENABLE_ACTIONS` gesetzt, wird der Wert in `dozzle.yml` ignoriert und der Assistent zeigt den Schalter als gesperrt an. Um eine Einstellung wieder über den Assistenten zu verwalten, entferne die Variable aus deiner Compose-Datei.

## <Icon icon="mdi:shield-lock-outline" inline /> Sicherheit

- **Der Login ist der erste Schritt.** Ein Neustart nach dem Speichern eines Kontos oder Proxys schaltet den Login ein, bevor irgendeine andere Einstellung geändert werden kann.
- **Nur ein angemeldeter Benutzer kann Aktionen und Shell ändern oder Dozzle neu starten.** Der Benutzer braucht alle Rollen.
- **Ohne Login gibt es ein Zeitfenster von 15 Minuten.** Ist `authProvider` auf `none` gesetzt, lassen sich diese Einstellungen nur innerhalb von 15 Minuten nach dem Start von Dozzle ändern. Danach ist der Assistent schreibgeschützt, bis du den Login einschaltest oder Dozzle neu startest.
- **Routen werden weiterhin beim Start festgelegt.** Der Assistent schreibt nur in `dozzle.yml`. Die Endpunkte für Aktionen und Shell werden beim Start von Dozzle registriert, genau wie bei Umgebungsvariablen, also wird nichts aktiviert, bevor Dozzle neu startet.
