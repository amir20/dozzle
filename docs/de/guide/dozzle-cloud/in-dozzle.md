---
title: In deinem Dozzle
description: Was sich in deinem eigenen Dozzle ändert, wenn du es mit Dozzle Cloud verbindest, darunter Ask Dozzle, Metrik-Verlauf und Alarme, die ein Neuladen überstehen.
sourceHash: af42207c220c
---

# In deinem Dozzle

Was sich im eigenen Dozzle ändert, sobald eine Instanz [verbunden](/de/guide/dozzle-cloud/connecting) ist. Alles hier lebt in der Dozzle-Oberfläche, die du ohnehin benutzt, direkt neben den Logs, um die es geht, und nicht auf einer separaten Website.

## Lokale Funktionen werden nie gesperrt

Dozzle verliert nichts, wenn Cloud nicht eingerichtet ist. Alarme lösen weiterhin aus, erscheinen weiterhin im Log-Stream und erreichen weiterhin deine Webhooks. Was die Verbindung hinzufügt, ist **Gedächtnis**: derselbe Alarm ist nach einem Neuladen, einem Neustart und auch eine Woche später noch da.

Das ist die ganze Trennlinie zwischen beiden. Dozzle gehört der Container, der gerade vor dir liegt, und Dozzle kann auf ihm handeln. Cloud gehört alles, was Vergangenheit ist, was mehrere Instanzen umspannt und was zum Konto gehört.

Eine Installation ohne Cloud zeigt deshalb einen leeren Verlaufsbereich mit einer Zeile, die sagt, was dort stünde, und keine gesperrte Karte:

> Alarme erscheinen hier, sobald diese Instanz mit Dozzle Cloud verbunden ist. Bis dahin tauchen sie im Log-Stream auf und sind nach dem Neuladen vergessen.

## Die Dozzle-Cloud-Schublade

Dozzle fragen, Metriken und Alarme sind Reiter in einer Schublade, die sich von rechts öffnet. Es gibt sie nur, wenn Cloud verbunden ist, dann aber auf jeder Seite.

Gefunden wird sie über einen Streifen mit Symbolen am rechten Rand: jedes Symbol öffnet seinen Reiter in der Schublade. Ob der Streifen ausgeblendet ist, wird gemerkt, und ein kleiner Reiter am Rand holt ihn zurück, so wie sich die Seitenleiste auf der anderen Seite einklappt. Auf dem Telefon gibt es keinen Streifen, dort öffnet sich die Schublade bildschirmfüllend über die Werkzeugleiste oder die Befehlspalette.

Solange die Schublade offen ist, verdeckt sie die Logs. Alles darin, was zurück auf den Stream zeigt (die Zeilen anzeigen, ein Zeitpunkt in einem Diagramm), schließt sie deshalb zuerst und scrollt dann den Stream dorthin. <kbd>Esc</kbd> schließt sie.

Metriken und Alarme beziehen sich auf die Container auf dem Bildschirm, deshalb erscheinen sie nur auf einer Seite, die einen zeigt. Auf der Startseite oder in den Einstellungen bietet die Schublade nur den Chat an.

### <Icon icon="mdi:message-outline" inline /> Dozzle fragen

Eine Frage zu dem, was du gerade ansiehst. In einer Log-Ansicht wird sie anhand der Logs darin beantwortet, auf jeder anderen Seite anhand der gesamten Instanz. Über dem Eingabefeld steht, was mit der Frage mitgeht: welche Container oder welcher Host in der Ansicht sind, wie viele Zeilen, und die Log-Zeile, auf die du gezeigt hast, falls du aus dem Menü einer Zeile gekommen bist. Nichts wird still mitgeschickt.

Antworten enden in Dozzle statt in einem Link nach außen. Geht es um einen Zeitpunkt, führt die Aktion deinen eigenen Stream dorthin.

Öffnen über die Schaltfläche **Dozzle fragen** neben der Suchleiste, mit <kbd>⇧</kbd><kbd>⌘</kbd><kbd>K</kbd> auf jeder Seite, mit <kbd>.</kbd> in den Logs eines Containers, über das Container-Menü oder über eine Log-Zeile. In der <kbd>⌘</kbd><kbd>K</kbd>-Palette steht bei einer Eingabe, die wie eine Frage klingt, **Fragen** an erster Stelle, sodass <kbd>Enter</kbd> sie abschickt, und <kbd>⇧</kbd><kbd>⌘</kbd><kbd>↵</kbd> fragt aus jeder Zeile heraus. Unter Windows und Linux steht <kbd>Ctrl</kbd> für <kbd>⌘</kbd>.

### <Icon icon="mdi:chart-line" inline /> Metriken

Dozzle hält 300 Messpunkte im Browser und nichts dahinter, „war das vor einer Stunde auch schon so?“ hat lokal also keine Antwort. Dieses Panel liest die Messpunkte zurück, die deine Instanz die ganze Zeit übermittelt hat: CPU, Speicher, Netzwerk und Datenträger über die letzten 1 h, 6 h oder 24 h. Jeder Balken ist ein Durchschnitt, hinter dem der höchste Messpunkt blass gezeichnet ist, sodass eine kurze Spitze sichtbar bleibt, egal wie lang das Fenster ist, und wer über ein beliebiges Diagramm fährt, liest alle zum selben Zeitpunkt aus. Netzwerk und Datenträger summieren über das Fenster, was hinausging und was hereinkam. Pro ergänzt ein Fenster von 7 Tagen.

### <Icon icon="mdi:bell-outline" inline /> Alarme

Was auf den Containern der aktuellen Ansicht ausgelöst hat. Die Seite [Benachrichtigungen](/de/guide/alerts-and-webhooks) beantwortet dieselbe Frage für die gesamte Instanz; dieser Reiter ist auf das begrenzt, was auf dem Bildschirm steht, und genau deshalb hat er seinen Platz in Dozzle verdient. Beim Öffnen verschwindet der Punkt für ungesehene Alarme an der Glocke.

## Alarme, die ein Neuladen überstehen

Sobald Alarme gespeichert werden, tauchen sie an drei weiteren Stellen auf:

- **An der Regel, die sie ausgelöst hat**, sodass eine vor Monaten geschriebene Regel daran gemessen werden kann, was sie tatsächlich gefangen hat.
- **Als Punkt in der Container-Zeile** in der Container-Tabelle, eingefärbt nach Schweregrad. Ein Klick öffnet ein kleines Panel mit der Überschrift, dem Level, dem Zeitpunkt, der Zahl der zusammengefassten Ereignisse und einer Zusammenfassung in einer Zeile.
- **In der Aktivitätsliste** auf der Benachrichtigungsseite, filterbar und noch lange lesbar, wenn die Einblendung längst weg ist.

## „Zeig mir die Zeilen“

Jede dieser Oberflächen endet in derselben Aktion, und sie ist dort immer die primäre.

**Zeig mir die Zeilen** öffnet die historische Ansicht des Containers, gescrollt auf den Moment, den der Alarm oder das Ergebnis beschreibt, mit hervorgehobener Zeile und bereits ausgefülltem Suchbegriff. Alarme auf Metriken und Ereignisse tragen keine Log-Zeile, sie landen deshalb auf dem Zeitpunkt statt auf einer Zeile.

Das ist der eine Schritt, den Dozzle gehen kann und Cloud nicht, und genau deshalb liegen diese Panels überhaupt in Dozzle. Links nach außen bleiben dem vorbehalten, was Cloud wirklich besser kann: Abrechnung und API-Schlüssel, das Berichtsarchiv, Auswertungen über mehrere Instanzen und vollständige Untersuchungsprotokolle.

## Ergebnisse

Ergebnisse sind keine Alarme. Ein Alarm ist eine Regel, die du geschrieben hast; ein Ergebnis ist etwas, das Cloud bemerkt hat, ohne dass eine Regel es abdeckt. Beide werden nie nebeneinander aufgelistet, und die Ergebnis-Oberfläche existiert nur, wenn Cloud eingerichtet ist, statt als dauerhafte Werbung in der Navigation zu stehen.

Was Ergebnisse in welchem Tarif abdecken, steht unter [Dozzle Cloud](/de/guide/dozzle-cloud).

## Wieder abschalten

Das Trennen der Instanz entfernt jede Oberfläche von dieser Seite und lässt Dozzle genau so zurück, wie es vorher war: Alarme lösen weiter aus, erscheinen weiter im Stream und sind nach dem Neuladen vergessen. Was dann nicht mehr deinen Host verlässt, steht unter [Deine Daten](/de/guide/dozzle-cloud/your-data).
