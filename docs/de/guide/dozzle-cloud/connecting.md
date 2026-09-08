---
title: Instanz verbinden
sourceHash: d6f8f7b6b845
---

# Instanz verbinden

Ein selbst gehostetes Dozzle mit [Dozzle Cloud](/de/guide/dozzle-cloud) verbinden, prüfen ob es wirklich verbunden ist, und beheben wenn nicht.

## Eine Instanz verbinden

1. Öffne dein selbst gehostetes Dozzle und klicke oben in der Leiste auf das **Cloud**-Symbol.
2. Klicke auf **Link instance**. Du wirst zu Cloud geleitet, um dich mit GitHub oder Google anzumelden und zu bestätigen.
3. Die Instanz erscheint innerhalb weniger Sekunden im Cloud-Dashboard.

Es gibt kein Passwort anzulegen und keinen Agent auf dem Host zu installieren.

## Du brauchst keine öffentliche IP, keinen offenen Port und keine Domain

Das ist die häufigste Sorge, und die Antwort ist bei allen dreien nein.

Deine Dozzle-Instanz öffnet eine **ausgehende** Verbindung zu Cloud und hält sie offen. Cloud verbindet sich nie zurück zu dir, sucht nie nach deinem Host und muss deine Adresse nie erreichen. Damit funktioniert es ganz normal, wenn Dozzle:

- hinter NAT im Heimnetz läuft, ohne Portweiterleitung
- eine private RFC1918-Adresse wie `192.168.1.50` hat
- in einem Tailscale-, WireGuard- oder ZeroTier-Netz läuft
- hinter CGNAT liegt, wo du gar nicht portweiterleiten könntest
- auf einem Laptop läuft, der das Netzwerk wechselt

Es braucht keinen Reverse Proxy, keinen dynamischen DNS-Namen und keine feste IP.

## Firewall-Regeln

Nur **ausgehender** Zugriff ist nötig. Erlaube deinem Dozzle-Host:

```
agent.doligence.dozzle.dev:443    (TCP, outbound)
```

Dieses eine Ziel auf Port 443 reicht. Wenn deine Firewall nach Hostname statt IP filtert, erlaube den Hostnamen — die Adressen dahinter können sich ändern. Die meisten Heim- und Büro-Firewalls erlauben ausgehenden Verkehr ohnehin, meistens gibt es also gar nichts einzurichten.

## Wo Regeln und Kanäle liegen

Darüber stolpern fast alle, deshalb einmal deutlich:

| Was du ändern willst                                              | Wo du es tust                                                      |
| ----------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Was einen Alarm auslöst** — Container, Muster, Schwellwerte     | Selbst gehostetes Dozzle → [Alarme](/de/guide/alerts-and-webhooks) |
| **Wohin Alarme zugestellt werden** — E-Mail, Telegram, Slack, ... | Dozzle Cloud → [Kanäle](/de/guide/dozzle-cloud/channels)           |
| Alte Alarme ansehen, stummschalten, Tarif wechseln                | Dozzle Cloud                                                       |

Die Regel wird auf deiner eigenen Instanz definiert, weil dort deine Logs liegen. Die Zustellung wird in Cloud konfiguriert, weil dort die Verbindung zu deinem Handy gehalten wird. Wenn du in Cloud nach einer Stelle suchst, um „sag mir Bescheid, wenn dieser Container Fehler wirft" einzustellen, und sie nicht findest: das ist der Grund. Öffne stattdessen dein selbst gehostetes Dozzle.

## Mehr als eine Instanz verbinden

Jede Instanz wird einzeln verbunden, mit denselben Schritten und demselben Cloud-Konto. Danach erscheinen alle zusammen im Dashboard, und Fragen im Chat decken jede verbundene Instanz auf einmal ab.

Diese gemeinsame Ansicht lebt in Cloud, nicht in einem einzelnen selbst gehosteten Dozzle. Ein selbst gehostetes Dozzle zeigt die Hosts, die du direkt darauf konfiguriert hast; andere verbundene Instanzen zeigt es nicht.

Mehrere Dozzle-Instanzen zu verbinden ist etwas anderes als Dozzles eigene Funktionen [Agent-Modus](/de/guide/agent) und [Remote-Hosts](/de/guide/remote-hosts), die zusätzliche Docker-Hosts an ein einzelnes Dozzle anbinden. Beides wird unterstützt und lässt sich kombinieren.

Der kostenlose Tarif verbindet eine Instanz gleichzeitig. Siehe [Tarife & Limits](/de/guide/dozzle-cloud/plans).

## Es taucht nichts auf

Arbeite das der Reihe nach ab.

**1. Läuft der Dozzle-Container?**
Wenn Dozzle selbst gestoppt ist oder neu startet, erreicht Cloud nichts.

**2. Wurde die Verbindung überhaupt abgeschlossen?**
Den Vorgang zu starten und nicht zu bestätigen hinterlässt nichts. Wiederhole die Schritte oben und prüfe, ob die Instanz dann auf der Instances-Seite auftaucht.

**3. Wurde der API-Schlüssel gelöscht?**
Das Löschen des API-Schlüssels trennt die Instanz dauerhaft. Der alte Schlüssel lässt sich nicht wieder anhängen — verbinde erneut, um einen neuen zu bekommen.

**4. Wird ausgehender Verkehr blockiert?**
Restriktive Netze (Unternehmen, Universität, manche VPS-Anbieter) blockieren ausgehendes 443 zu Zielen, die nicht auf einer Allowlist stehen. Siehe _Firewall-Regeln_ oben.

**5. Hast du das Instanz-Limit im kostenlosen Tarif erreicht?**
Der kostenlose Tarif verbindet eine Instanz gleichzeitig. Beim Versuch, eine zweite zu verbinden, erscheint eine Limit-Meldung statt einer Verbindung.

**6. Ist der Container von der Weiterleitung ausgenommen?**
Ein Container mit dem Label `dev.dozzle.cloud.min_level=disabled` sendet absichtlich nichts. Wenn ein bestimmter Container fehlt und andere funktionieren, prüfe seine Labels. Siehe [Deine Daten](/de/guide/dozzle-cloud/your-data).

## Den Agent Container steuern lassen

Logs und Containerzustand lesen funktioniert, sobald eine Instanz verbunden ist. Starten, Stoppen und Neustarten lehnt deine Instanz ab, solange du es nicht selbst aktivierst:

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

Das ist eine Einstellung auf **deinem** Dozzle, nicht in Cloud, denn sie regelt, was dein Dozzle mit deinen Containern zu tun bereit ist. Starte Dozzle nach der Änderung neu. Siehe [Aktionen](/de/guide/actions).

## Verbindung trennen

Lösche den API-Schlüssel der Instanz auf der Instances-Seite in Cloud. Die Verbindung fällt weg, es werden keine Daten mehr weitergeleitet, und dein selbst gehostetes Dozzle läuft genau wie vorher weiter. Das Verbinden ändert das lokale Betrachten der Logs nie.
