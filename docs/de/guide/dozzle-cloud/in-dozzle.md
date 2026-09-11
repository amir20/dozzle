---
title: In deinem Dozzle
sourceHash: 8053f9b40bcd
---

# In deinem Dozzle

Was sich im eigenen Dozzle ändert, sobald eine Instanz [verbunden](/de/guide/dozzle-cloud/connecting) ist. Alles hier lebt in der Dozzle-Oberfläche, die du ohnehin benutzt, direkt neben den Logs, um die es geht, und nicht auf einer separaten Website.

## Lokale Funktionen werden nie gesperrt

Dozzle verliert nichts, wenn Cloud nicht eingerichtet ist. Alarme lösen weiterhin aus, erscheinen weiterhin im Log-Stream und erreichen weiterhin deine Webhooks. Was die Verbindung hinzufügt, ist **Gedächtnis**: derselbe Alarm ist nach einem Neuladen, einem Neustart und auch eine Woche später noch da.

Das ist die ganze Trennlinie zwischen beiden. Dozzle gehört der Container, der gerade vor dir liegt, und Dozzle kann auf ihm handeln. Cloud gehört alles, was Vergangenheit ist, was mehrere Instanzen umspannt und was zum Konto gehört.

Eine Installation ohne Cloud zeigt deshalb einen leeren Verlaufsbereich mit einer Zeile, die sagt, was dort stünde, und keine gesperrte Karte:

> Alarme erscheinen hier, sobald diese Instanz mit Dozzle Cloud verbunden ist. Bis dahin tauchen sie im Log-Stream auf und sind nach dem Neuladen vergessen.

## Die Cloud-Leiste

Ein Streifen mit Symbolen am rechten Rand der Log-Ansicht und das Panel, das eines davon öffnet. Sie wird nur eingebunden, wenn Cloud verbunden ist, und nur in einer Ansicht, in der tatsächlich Logs stehen; auf der Startseite oder in den Einstellungen erscheint sie also nie.

Das Panel liegt **neben** dem Stream statt darüber: die Seite reserviert genau seine Breite, dadurch verdeckt nichts auf der Leiste je die Zeilen, über die es spricht. Auf dem Telefon ist für einen dauerhaften Streifen kein Platz, dort wird das Panel zu einer bildschirmfüllenden Ebene, die sich über die Werkzeugleiste oder die Befehlspalette öffnet.

Ob die Leiste ausgeblendet ist, wird gemerkt. Ein kleiner Reiter am Rand holt sie zurück, so wie sich die Seitenleiste auf der anderen Seite einklappt.

### <Icon icon="mdi:message-outline" inline /> Dozzle fragen

Eine Frage zu der Ansicht, die gerade offen ist, beantwortet anhand der Logs darin. Über dem Eingabefeld steht, was mit der Frage mitgeht: welche Container in der Ansicht sind, wie viele Zeilen, und die Log-Zeile, auf die du gezeigt hast, falls du aus dem Menü einer Zeile gekommen bist. Nichts wird still mitgeschickt.

Antworten enden in Dozzle statt in einem Link nach außen. Geht es um einen Zeitpunkt, führt die Aktion deinen eigenen Stream dorthin.

Öffnen mit <kbd>Shift</kbd> + <kbd>⌘</kbd> + <kbd>K</kbd>, über das Container-Menü oder über eine Log-Zeile.

### <Icon icon="mdi:chart-line" inline /> Metriken

Dozzle hält 300 Messpunkte im Browser und nichts dahinter, „war das vor einer Stunde auch schon so?“ hat lokal also keine Antwort. Dieses Panel liest die Messpunkte zurück, die deine Instanz die ganze Zeit übermittelt hat: CPU und Speicher über die letzten 1 h, 6 h oder 24 h, mit hervorgehobenem Höchstwert; beim Überfahren liest das Diagramm sich selbst vor. Pro ergänzt ein Fenster von 7 Tagen.

### <Icon icon="mdi:bell-outline" inline /> Alarme

Was auf den Containern der aktuellen Ansicht ausgelöst hat. Die Seite [Benachrichtigungen](/de/guide/alerts-and-webhooks) beantwortet dieselbe Frage für die gesamte Instanz; dieses Panel ist auf das begrenzt, was auf dem Bildschirm steht, und genau deshalb hat es seinen Platz neben dem Stream verdient. Beim Öffnen verschwindet der Punkt an der Glocke.

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
