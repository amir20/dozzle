---
title: Tarife & Limits
sourceHash: be1bd7a795e4
---

# Tarife & Limits

Tarife, Preise und Kontingente stehen auf der [Preisseite von Dozzle Cloud](https://cloud.dozzle.dev/pricing), die immer aktuell ist. Diese Seite beschreibt, wie die Limits aus deiner Sicht aussehen, wenn du an sie stößt.

## Alarme sind plötzlich roh und repetitiv

Höchstwahrscheinlich hast du dein monatliches Ereignis-Kontingent überschritten. Es geht nichts kaputt: Die Ereignis-Historie wird weiter aufgezeichnet, aber die Auswertung pausiert, ungefähr jedes zehnte Ereignis kommt als roher Alarm durch, und Wiederholungen werden nicht mehr zu einem Alarm mit Zähler gebündelt. Prüfe zuerst die Nutzungsseite in Cloud. Kontingente werden zu Monatsbeginn zurückgesetzt, und das gilt auch in bezahlten Tarifen.

## Eine Suche findet nichts von letzter Woche

Die Suche reicht nur so weit zurück wie die Aufbewahrung deines Tarifs. Alles Ältere ist bereits gelöscht, auch wenn es passiert ist. Fragst du nach einem längeren Metrik-Zeitraum als dein Tarif erlaubt, bekommst du den Zeitraum, den du tatsächlich hast, statt einen Fehler.

## Das Instanz-Limit

Der kostenlose Tarif verbindet eine Instanz gleichzeitig. Beim Verbinden einer zweiten erscheint eine Limit-Meldung. Du hast zwei Möglichkeiten:

- **Den Platz umziehen.** Lösche den API-Schlüssel der bestehenden Instanz auf der Instances-Seite und verbinde dann die neue. Für die alte Instanz ist das endgültig. Ihre Historie bleibt, aber du müsstest sie von vorn neu verbinden.
- **Upgraden**, um beide gleichzeitig verbunden zu lassen.

## Nutzung prüfen

Die Nutzungsseite in Cloud zeigt Ereignisse, Log-Bytes und Assistenz-Chats dieses Monats gegen dein Kontingent. Du kannst auch im Chat fragen: „wie viel habe ich diesen Monat verbraucht?".

## Tarif wechseln oder kündigen

Upgrade über die Preisseite oder die Einstellungen. Die Abrechnung läuft über Stripe; Zahlungsmethoden, Rechnungen und Belege verwaltest du dort über den Billing-Link in deinen Einstellungen.

Eine Kündigung stoppt künftige Abbuchungen und stellt dich zum Ende des bezahlten Zeitraums auf den kostenlosen Tarif um. Konto und Historie bleiben erhalten.
