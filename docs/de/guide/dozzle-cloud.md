---
title: Dozzle Cloud
sourceHash: ddf1502ea23d
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) ist ein optionaler verwalteter Begleiter für selbst gehostetes Dozzle. Dozzle selbst bleibt vollständig quelloffen und selbst gehostet; Cloud setzt darauf auf und übernimmt den Teil, der wirklich schwer selbst zu betreiben ist: die Entscheidung, was dich aufwecken darf, und die Frage, was tatsächlich kaputtgegangen ist.

Dein Dozzle baut eine ausgehende Verbindung zu Cloud auf. Es gibt keinen eingehenden Port, keine öffentliche IP und keinen Agent zu installieren.

Der kostenlose Tarif ist die komplette Alarmierungsebene: Jeder Alarm, den deine Dozzle-Regeln auslösen, wird zu einer lesbaren Nachricht ausgewertet, Wiederholungen werden zu einem Alarm mit Zähler gebündelt, und er geht per E-Mail, Telegram, Discord, Slack, ntfy, Webhook oder Browser-Push raus. Die bezahlten Tarife fügen die proaktive Hälfte hinzu: Cloud liest jeden Morgen deine Logs und meldet Probleme, bei denen nie ein Alarm angeschlagen hat, dazu kommen längere Historie und mehr Instanzen.

Die ganze Tour steht unter [Features](https://cloud.dozzle.dev/features), was jeder Tarif enthält unter [Preise](https://cloud.dozzle.dev/pricing).

## Wie es weitergeht

| Seite                                                      | Worum es geht                                                                                 |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Instanz verbinden](/de/guide/dozzle-cloud/connecting)     | Verbinden, warum keine öffentliche IP und kein offener Port nötig sind, Firewall, Fehlersuche |
| [In deinem Dozzle](/de/guide/dozzle-cloud/in-dozzle)       | Die Cloud-Leiste, Alarme die ein Neuladen überstehen, und was ohne Cloud weiter funktioniert  |
| [Benachrichtigungskanäle](/de/guide/dozzle-cloud/channels) | Alle Kanäle, wie du jeden einrichtest, und wie du es leiser bekommst                          |
| [Tarife & Limits](/de/guide/dozzle-cloud/plans)            | An ein Limit stoßen, das Instanz-Limit, Nutzung, Kündigung                                    |
| [Deine Daten](/de/guide/dozzle-cloud/your-data)            | Was deinen Host verlässt, wie du das stoppst, was Cloud speichert, API-Schlüssel              |

Die Alarmregeln selbst werden auf deiner eigenen Instanz konfiguriert, nicht in Cloud. Siehe [Alarme](/de/guide/alerts-and-webhooks).

## Feedback

Dozzle Cloud wird von derselben Person gebaut wie Dozzle, und der Anspruch ist derselbe: Dinge, die Leute wirklich benutzen wollen. Wenn du es ausprobierst und etwas sich falsch anfühlt, fehlt oder echt nützlich ist, [eröffne bitte eine Discussion](https://github.com/amir20/dozzle/discussions). Dieses Feedback bestimmt, was als Nächstes gebaut wird.
