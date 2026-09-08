---
title: Dozzle Cloud
sourceHash: 49197a749322
---

# Dozzle Cloud

[Dozzle Cloud](https://cloud.dozzle.dev) ist ein optionaler verwalteter Begleiter für selbst gehostetes Dozzle. Dozzle selbst bleibt vollständig quelloffen und selbst gehostet; Cloud setzt darauf auf und übernimmt den Teil, der wirklich schwer selbst zu betreiben ist: die Entscheidung, was dich aufwecken darf, und die Frage, was tatsächlich kaputtgegangen ist.

Dein Dozzle baut eine ausgehende Verbindung zu Cloud auf. Es gibt keinen eingehenden Port, keine öffentliche IP und keinen Agent zu installieren.

**Kostenlos hält dir den Rücken frei. Pro geht selbst auf die Suche.**

## <Icon icon="mdi:bell-ring-outline" inline /> Kostenlos: eine intelligente Benachrichtigungsebene

Die meisten Log-Alarme sind ein Regex plus ein Webhook. Beim ersten Crash-Loop werden daraus zweihundert identische Nachrichten und du schaltest den Kanal stumm. Genau diesen Teil behebt die kostenlose Stufe, und sie ist das vollständige Alarmierungsprodukt, nicht dessen Testversion.

- **Intelligente Alarme** — jeder Alarm, den deine Dozzle-Regeln auslösen, wird zu einem Satz aufbereitet, der die Ursache, den Container und den Schweregrad nennt, mit einem Link zurück auf genau die Log-Zeile in deinem eigenen Dozzle.
- **Wiederholungen werden gebündelt** — 47 Abstürze kommen als ein Alarm an, der 47 sagt. Kommt der Container wieder hoch, bekommst du eine Entwarnung.
- **Standardmäßig ruhig** — Unterdrückung, Schweregrad-Filter und musterbasiertes Stummschalten auf jedem Kanal. Stelle _diese Art von Alarm_ stumm statt nur diesen einen; alles wirklich Neue kommt weiterhin durch.
- **Alle Kanäle** — E-Mail, Telegram, Discord, Slack, ntfy, Webhooks und Browser-Push, alle im kostenlosen Tarif. Siehe [Benachrichtigungskanäle](/de/guide/dozzle-cloud/channels).
- **Suche und Metriken inklusive** — jedes Ereignis ist ab dem Moment durchsuchbar, in dem es ankommt, und CPU, Speicher, Netzwerk und Festplatte werden als Historie aufgezeichnet. Beides zählt nicht auf dein Ereigniskontingent.
- **Ein Befund pro Woche** — auch kostenlos liest Cloud deine Logs und zeigt dir das Schwerwiegendste, worauf kein Alarm angeschlagen hat.
- **Eine funktionierende Standardregel** — beim Verbinden einer Instanz wird eine für dich angelegt (Container, die mit einem Fehler beenden), sodass ein neues Konto schon am ersten Tag einen nützlichen Alarm bekommt, ganz ohne Konfiguration.
- **Chat-Agent und MCP** — frag in Telegram oder Discord „gab es heute Fehler?" und starte, stoppe oder starte einen Container aus derselben Unterhaltung neu, sobald du [Aktionen](/de/guide/actions) auf deiner Instanz aktiviert hast. Der MCP-Zugriff ist in jedem Tarif unbegrenzt.

> [!TIP]
> Eine frisch verbundene Instanz bekommt 7 Tage lang das volle Pro-Erlebnis: jeden Morgen jeden Befund. Danach pendelt sich der kostenlose Tarif auf einen Befund pro Woche ein.

## <Icon icon="mdi:robot-outline" inline /> Pro: es sucht, bevor überhaupt etwas Alarm schlägt

Kostenlos sagt dir, _dass_ etwas passiert ist, und bleibt still, wenn nichts war. Pro ist die Hälfte, die nicht darauf wartet, dass ein Alarm existiert.

- **Proaktive Auswertung, jeden Morgen** — Cloud liest deine Fehler-Logs, fasst sie zu Mustern zusammen und meldet, was behoben werden sollte. Genau hier taucht eine langsam vollaufende Festplatte oder ein still vor sich hin neustartender Container auf, an einem Tag, an dem gar nichts ausgelöst hat. Dafür muss keine Alarmregel existieren.
- **Jeder Befund, täglich, mit der Lösung** — nicht einer pro Woche und der Rest gesperrt. Befunde altern von Tag zu Tag, solange das Problem besteht („passiert immer noch, Tag vier, dreimal schlimmer"), und schließen sich selbst, wenn es aufhört.
- **Auswertung, die nachschaut** — wenn der Alarmtext allein nicht reicht, inspiziert sie den Container und liest die umgebenden Logs, bevor sie entscheidet, statt zu raten.
- **Vollständige Untersuchungen auf Knopfdruck** — ein Klick startet mehr Durchläufe mit einem stärkeren Modell, korreliert über deine Container, Hosts und den Zeitverlauf hinweg und liefert eine Ursache samt konkreter Schritte zurück.
- **Jeder Host, ein Dashboard** — verbinde so viele Dozzle-Instanzen, wie du betreibst. Fragen im Chat decken alle auf einmal ab.
- **Längeres Gedächtnis** — 30 Tage durchsuchbare Logs und Metriken statt 24 Stunden. Das ist der Unterschied zwischen „was ist heute Nacht passiert" und „passiert das schon den ganzen Monat".

Die vollständige Gegenüberstellung steht unter [Tarife & Limits](/de/guide/dozzle-cloud/plans).

## Wie es weitergeht

| Seite                                                      | Worum es geht                                                                                 |
| ---------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| [Instanz verbinden](/de/guide/dozzle-cloud/connecting)     | Verbinden, warum keine öffentliche IP und kein offener Port nötig sind, Firewall, Fehlersuche |
| [Benachrichtigungskanäle](/de/guide/dozzle-cloud/channels) | Alle Kanäle, wie du jeden einrichtest, und wie du es leiser bekommst                          |
| [Tarife & Limits](/de/guide/dozzle-cloud/plans)            | Was jeder Tarif enthält, was ein ausgewertetes Ereignis ist, was beim Überschreiten passiert  |
| [Deine Daten](/de/guide/dozzle-cloud/your-data)            | Was deinen Host verlässt, wie du das stoppst, was Cloud speichert, API-Schlüssel              |

Die Alarmregeln selbst werden auf deiner eigenen Instanz konfiguriert, nicht in Cloud. Siehe [Alarme](/de/guide/alerts-and-webhooks).

## Feedback

Dozzle Cloud wird von derselben Person gebaut wie Dozzle, und der Anspruch ist derselbe: Dinge, die Leute wirklich benutzen wollen. Wenn du es ausprobierst und etwas sich falsch anfühlt, fehlt oder echt nützlich ist, [eröffne bitte eine Discussion](https://github.com/amir20/dozzle/discussions). Dieses Feedback bestimmt, was als Nächstes gebaut wird.
