---
title: Tarife & Limits
sourceHash: f7c6bbdc83ea
---

# Tarife & Limits

Was jeder Tarif enthält, was darauf angerechnet wird, und was beim Überschreiten passiert.

Der kostenlose Tarif ist das vollständige Alarmierungsprodukt, nicht dessen Testversion. Was die bezahlten Tarife kaufen, ist die proaktive Hälfte — die Durchsicht, die deine Logs liest und findet, worauf nie ein Alarm angeschlagen hat — plus Luft nach oben.

## Tarife

|                                                           |  Kostenlos   |      Pro      |     Team      |
| --------------------------------------------------------- | :----------: | :-----------: | :-----------: |
| Preis                                                     |     0 $      |  5 $ / Monat  | 15 $ / Monat  |
| Befunde (liest deine Logs, findet was nie Alarm auslöste) |  1 / Woche   | Alle, täglich | Alle, täglich |
| Lösung zu jedem Befund                                    |      —       |       ✓       |       ✓       |
| Auswertung prüft Container und Logs im Zweifel            |      —       |       ✓       |       ✓       |
| Vollständige Untersuchung auf Knopfdruck                  |      —       |       ✓       |       ✓       |
| Ausgewertete Ereignisse pro Monat                         |    2.000     |      50K      |     250K      |
| Durchsuchbare Logs                                        | 10 GB · 24 h | 50 GB · 30 T  | 100 GB · 30 T |
| Alarm- und Ereignis-Historie                              |    1 Tag     |    14 Tage    |    30 Tage    |
| Metrik-Historie (CPU, Speicher, Netzwerk, Festplatte)     |     24 h     |    30 Tage    |    30 Tage    |
| Verbundene Instanzen                                      |      1       |  Unbegrenzt   |  Unbegrenzt   |
| Assistenz-Chats pro Monat                                 |      10      |      200      |     1.000     |
| Priorisierter Support                                     |      —       |       ✓       |       ✓       |

Intelligente Alarme, Bündelung von Wiederholungen, Unterdrückung und Schweregrad-Filter, Suchindizierung, Metrik-Aufzeichnung, alle Benachrichtigungskanäle, Container-Aktionen und unbegrenzter MCP-Zugriff sind in **jedem** Tarif enthalten, auch im kostenlosen.

Die aktuellen Preise stehen auf [cloud.dozzle.dev](https://cloud.dozzle.dev).

## Was als ausgewertetes Ereignis zählt

Ein **ausgewertetes Ereignis** ist ein Container-Ereignis oder eine passende Log-Zeile, die durch die Auswertungs-Pipeline gelaufen ist, welche entscheidet, ob ein neuer Alarm rausgeht, ob es in einen bestehenden gebündelt wird oder ob es still bleibt. Du bezahlst für diese Arbeit, nicht für rohen Speicher.

Gewöhnliche Log-Zeilen sind keine Ereignisse. Sie zählen stattdessen auf das durchsuchbare Log-Volumen. Ein sehr geschwätziger Container kostet also Speicher, während ein Container im Crash-Loop Ereignisse kostet.

Beendet ein Container 47-mal, sind das 47 Ereignisse gegen das Limit — aber ein Alarm, der 47 sagt. Genau darum geht es.

**Befunde kosten keines von beidem.** Die Log-Durchsicht liest deine Logs statt deiner Alarm-Historie, berührt also den Ereigniszähler nicht und funktioniert sogar ganz ohne konfigurierte Alarmregeln. Es zählt nur dein monatliches Log-Volumen.

**Suche und Metriken sind in jedem Tarif kostenlos.** Die Suchindizierung läuft ab dem Moment, in dem eine Instanz verbunden ist, und CPU-, Speicher-, Netzwerk- und Festplattenreihen fließen ohne Aufpreis von deinen Instanzen. Der Tarif ändert nur, wie viel und wie weit zurück.

## Über dem Kontingent

Es geht nichts kaputt. Du landest im Sampling-Modus:

- Die Auswertung pausiert.
- Die Ereignis-Historie wird weiter aufgezeichnet, es geht also nichts verloren.
- Ungefähr jedes zehnte Ereignis kommt als **roher** Alarm durch, damit du weiter siehst, was los ist.
- Wiederholungen werden nicht mehr zu einem Alarm mit Zähler gebündelt.

Praktisch heißt das: Alarme werden lauter und weniger nützlich, statt zu verschwinden, und du merkst es im Posteingang, bevor du es auf einer Nutzungsseite siehst. Wenn deine Alarme plötzlich roh und repetitiv sind, prüfe zuerst die Nutzung.

Das gilt auch in bezahlten Tarifen. Kontingente werden zu Monatsbeginn zurückgesetzt.

> [!TIP]
> Die meisten Konten kommen nie in die Nähe. Nur ein echter Feuerwehrschlauch — ein Container, der tagelang im Crash-Loop hängt — geht darüber hinaus, und der Alarm, der diesen Container benennt, kommt lange vor dem Limit an.

## Aufbewahrung

Die Aufbewahrung bestimmt, wie weit deine Historie zurückreicht: Alarme, Ereignisse und Log-Suchergebnisse. Im kostenlosen Tarif ist das ein Tag, eine Suche nach etwas von letzter Woche liefert also nichts, obwohl es passiert ist. Das ist der häufigste Grund, warum eine Suche Daten scheinbar „verliert".

Die Metrik-Aufbewahrung beträgt 24 Stunden kostenlos und 30 Tage in den bezahlten Tarifen. Fragst du nach einem längeren Zeitraum als dein Tarif erlaubt, bekommst du den Zeitraum, den du tatsächlich hast, statt einen Fehler.

## Das Instanz-Limit

Der kostenlose Tarif verbindet eine Instanz gleichzeitig. Beim Verbinden einer zweiten erscheint eine Limit-Meldung. Du hast zwei Möglichkeiten:

- **Den Platz umziehen.** Lösche den API-Schlüssel der bestehenden Instanz auf der Instances-Seite und verbinde dann die neue. Für die alte Instanz ist das endgültig — ihre Historie bleibt, aber du müsstest sie von vorn neu verbinden.
- **Upgraden**, um beide gleichzeitig verbunden zu lassen.

Das Limit betrifft, wie viel ein kostenloses Konto weiterleitet, es ist keine Funktionssperre. Alles andere funktioniert kostenlos mit der Instanz, die du verbunden hast.

## Nutzung prüfen

Die Nutzungsseite in Cloud zeigt Ereignisse, Log-Bytes und Assistenz-Chats dieses Monats gegen dein Kontingent. Du kannst auch im Chat fragen: „wie viel habe ich diesen Monat verbraucht?".

## Tarif wechseln oder kündigen

Upgrade über die Preisseite oder die Einstellungen. Die Abrechnung läuft über Stripe; Zahlungsmethoden, Rechnungen und Belege verwaltest du dort über den Billing-Link in deinen Einstellungen.

Eine Kündigung stoppt künftige Abbuchungen und stellt dich zum Ende des bezahlten Zeitraums auf den kostenlosen Tarif um. Konto und Historie bleiben erhalten.
