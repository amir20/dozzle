---
title: Benachrichtigungskanäle
sourceHash: baae80ce439a
---

# Benachrichtigungskanäle

Kanäle werden in [Dozzle Cloud](/de/guide/dozzle-cloud) konfiguriert und steuern, _wohin_ Alarme gehen. Was einen Alarm _auslöst_, konfigurierst du auf deiner selbst gehosteten Instanz — siehe [Alarme](/de/guide/alerts-and-webhooks).

Aktiviere so viele du willst. Jeder aktivierte Kanal bekommt jeden Alarm, und jeder lässt sich unabhängig ein- und ausschalten.

## Verfügbare Kanäle

| Kanal                                                       | Alarme | Tägliche Zusammenfassung | Zwei-Wege-Agent |
| ----------------------------------------------------------- | :----: | :----------------------: | :-------------: |
| <Icon icon="mdi:email-outline" inline /> E-Mail             |   ✓    |            ✓             |                 |
| <Icon icon="mdi:telegram" inline /> Telegram                |   ✓    |            ✓             |        ✓        |
| <Icon icon="ic:baseline-discord" inline /> Discord-Bot (DM) |   ✓    |            ✓             |        ✓        |
| <Icon icon="ic:baseline-discord" inline /> Discord-Webhook  |   ✓    |            ✓             |                 |
| <Icon icon="mdi:slack" inline /> Slack                      |   ✓    |                          |                 |
| <Icon icon="simple-icons:ntfy" inline /> ntfy               |   ✓    |                          |                 |
| <Icon icon="mdi:webhook" inline /> Webhooks                 |   ✓    |                          |                 |
| <Icon icon="mdi:bell-badge-outline" inline /> Browser-Push  |   ✓    |                          |                 |

Alle Kanäle sind in jedem Tarif verfügbar, auch im kostenlosen.

## E-Mail

Wird automatisch mit der Adresse eingerichtet, mit der du dich registriert hast. Nichts zu konfigurieren. Zum Abschalten deaktivierst du den E-Mail-Kanal. Wenn Alarme unerwartet ausbleiben, schau zuerst in den Spam-Ordner — der erste Alarm landet gelegentlich dort, und ihn als „kein Spam" zu markieren behebt das dauerhaft.

## Telegram

Wähle auf der Channels-Seite **Telegram**, folge dem Link zum Bot und drücke **Start**. Der Kanal wird aktiv, sobald der Bot von dir gehört hat.

Telegram funktioniert in beide Richtungen. Du kannst im selben Chat antworten und nach deinen Containern fragen — „gab es heute Fehler?", „zeig mir die CPU-Auslastung", „welche Alarme habe ich?" — und bekommst Antworten zum Live-Zustand.

## Discord

Discord hat **zwei getrennte Kanaltypen**, und beide gleichzeitig laufen zu haben ist der übliche Grund dafür, jeden Alarm doppelt zu bekommen.

**Discord-Bot (Direktnachricht)** — der Bot schickt dir Alarme persönlich als DM. Zwei-Wege, du kannst ihm also Fragen stellen. Einrichtung durch Autorisieren des Bots auf der Channels-Seite.

**Discord-Webhook (Server-Kanal)** — Alarme werden in einen Kanal auf deinem Server gepostet, etwa `#alerts`. Nur eine Richtung. Einrichtung durch Anlegen eines Webhooks in den Server-Einstellungen und Einfügen der URL in Cloud.

Wenn Alarme sowohl in deinen DMs als auch in einem Server-Kanal ankommen, hast du beides konfiguriert. Deaktiviere, was du nicht willst; eines abzuschalten lässt das andere laufen. Ein häufiges Setup ist, den gemeinsamen Server-Kanal zu behalten und die DM abzuschalten.

## Slack

Lege in deinem Slack-Workspace einen Incoming Webhook an und füge die URL im Slack-Kanal auf der Channels-Seite ein.

## ntfy

Trage deine Topic-URL ein. Sowohl ntfy.sh als auch ein selbst gehosteter ntfy-Server funktionieren. Beliebt für Handy-Benachrichtigungen ohne zusätzliches Konto.

## Webhooks

Trage eine beliebige URL ein, die ein POST annimmt. Alarme werden als JSON zugestellt, du kannst sie also in alles leiten, was du ohnehin betreibst — Home Assistant, n8n, ein Skript, ein anderes Alarmierungswerkzeug.

> [!NOTE]
> Das ist ein Cloud-Kanal und etwas anderes als die Webhooks, die dein selbst gehostetes Dozzle direkt aufrufen kann. Die stehen in [Alarme](/de/guide/alerts-and-webhooks), samt den Go-Template-Variablen.

## Browser-Push

Aktiviere es auf der Channels-Seite und erlaube Benachrichtigungen, wenn dein Browser fragt. Alarme kommen dann als Desktop-Benachrichtigungen an.

Wenn danach nichts ankommt, hat der Browser die Berechtigung höchstwahrscheinlich verweigert. Browser fragen nach einer Ablehnung nicht erneut — lösche die Benachrichtigungs-Berechtigung der Seite in den Browser-Einstellungen und aktiviere sie neu. In einem privaten oder Inkognito-Fenster funktioniert Browser-Push nicht.

## <Icon icon="mdi:bell-sleep-outline" inline /> Für Ruhe sorgen

Du sollst nur gestört werden, wenn es zählt. Wenn Cloud laut ist, ist das ein Einstellungsproblem, und dafür gibt es diese Werkzeuge.

| Situation                                       | Tu das                                  |
| ----------------------------------------------- | --------------------------------------- |
| Ein wiederkehrender Fehler, den du schon kennst | **Muster stummschalten**                |
| Alarme sind nützlich, kommen aber zu oft        | **Daumen runter geben**                 |
| Geplante Wartung, Backups, Upgrades             | **Vor Beginn das Muster stummschalten** |
| Richtiger Alarm, falsche App                    | **Diesen Kanal deaktivieren**           |
| Du willst gar nichts mehr, von nirgendwo        | **Alle Kanäle deaktivieren**            |

Die Alarmregel zu löschen ist fast nie die richtige Antwort. Damit entfernst du eine ganze Überwachungskategorie, um eine laute Zeile loszuwerden.

### Einen wiederkehrenden Alarm stummschalten

Das Stummschalten ist musterbasiert: es bringt _diese Art von Alarm_ zum Schweigen, nicht nur den vor dir. Spätere Vorkommen bleiben still, und alles wirklich Andere kommt weiterhin durch.

- **Am Alarm** — öffne ihn in Cloud und wähle Stummschalten.
- **Im Chat** — sag „stell das stumm" oder „hör auf mir von X zu erzählen". Der Agent nennt genau das Muster, das er stummschalten will, und wartet auf deine Bestätigung, denn eine Stummschaltung ist dauerhaft und könnte später einen echten Ausfall verdecken.

Die Stummschaltung bleibt, bis du sie aufhebst. Frag „was habe ich stummgeschaltet?" für eine Liste, und heb sie genauso wieder auf. Stummgeschaltete Alarme werden weiterhin aufgezeichnet — Stummschalten ändert, was dich unterbricht, nicht was überwacht wird.

### Weniger, nicht gar nichts

Wenn ein Alarm wirklich nützlich ist, aber zu oft kommt, gib ihm **Daumen runter** statt ihn stummzuschalten. Das ist das Signal für „beobachte das weiter, unterbrich mich seltener". Daumen hoch für Alarme, die richtig lagen, hilft genauso.

### Wiederholungen werden bereits gebündelt

Prüfe vor dem Stummschalten, ob das Problem eigentlich Wiederholung ist. Wiederkehrende Vorkommen desselben Ausfalls werden zu einem einzigen Alarm mit Zähler zusammengefasst. Wenn du viele Alarme bekommst, sind es meist viele _verschiedene_ Probleme — oder du bist über dem Ereigniskontingent deines Tarifs, und die Alarme sind auf roh und ungebündelt zurückgefallen. Siehe [Tarife & Limits](/de/guide/dozzle-cloud/plans).

### An der Quelle filtern

Für einen Container, der im Normalbetrieb laut ist, liegt die bessere Lösung weiter oben: das Label `dev.dozzle.cloud.min_level` sorgt dafür, dass Zeilen niedriger Schwere deinen Host gar nicht erst verlassen. Siehe [Deine Daten](/de/guide/dozzle-cloud/your-data).

## Warum kam kein Alarm?

**1. Gibt es überhaupt eine Regel dafür?** Ein Fehler in deinen Logs erzeugt für sich genommen keinen Alarm; irgendetwas muss danach schauen. Die Standardregel deckt nur Container ab, die mit einem Fehler beenden — ein Container, der Fehler loggt und dabei weiterläuft, braucht eine Log-Regel.

**2. Ist ein Kanal aktiviert?** Eine Regel ohne aktivierten Kanal hat kein Ziel.

**3. Ist die Instanz verbunden?** War sie zum Zeitpunkt des Problems offline, wurde nichts weitergeleitet. Siehe [Instanz verbinden](/de/guide/dozzle-cloud/connecting).

**4. Wurde er in einen Alarm gebündelt, den du schon bekommen hast?** Vierzig Ausfälle erzeugen einen Alarm, der vierzig sagt. Das ist Absicht, kein verpasster Alarm.

**5. Hast du ihn stummgeschaltet?** Prüfe deine Stummschaltungsregeln.

**6. Ist der Container von der Weiterleitung ausgenommen?** Siehe [Deine Daten](/de/guide/dozzle-cloud/your-data).

**7. Bist du über den Limits deines Tarifs?** Jenseits des Kontingents ändert sich die Zustellung und Alarme werden gesampelt.

**8. Schau in den Spam-Ordner**, speziell bei E-Mail.
