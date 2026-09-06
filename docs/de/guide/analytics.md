---
title: Anonyme Analysedaten
sourceHash: 0fedd8c524b8
---

# Erhebung von Analysedaten

Dozzle erhebt über einen schlanken Beacon anonyme Nutzungsdaten, um Funktionen und Fehlerbehebungen besser priorisieren zu können. Es ist ein Open-Source-Projekt ohne Finanzierung, deshalb sind diese Daten das wichtigste Signal dafür, wo sich Aufwand lohnt.

## Was wird erhoben

Grob gesagt enthält der Beacon Dinge wie die Dozzle-Version, den Betriebsmodus (server, swarm, k8s, agent), den aktivierten Auth-Provider, einige Feature-Flags, die Version der Docker Engine und kleine Zählwerte (Anzahl der Hosts, Container, Filter). Zur Deduplizierung wird eine zufällige ID pro Installation mitgeschickt.

Log-Inhalte, Container-Namen, Image-Namen, IP-Adressen oder Nutzerkennungen werden niemals übertragen. Welche Felder genau enthalten sind, ändert sich mit der Zeit. Maßgeblich sind [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go) und der Sender in [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Wo werden die Daten gespeichert

Die Events gehen an `https://b.dozzle.dev/event`, einen kleinen Go-Dienst, der sie zur späteren Verarbeitung in eine Datei auf DigitalOcean schreibt.

## Deaktivieren

Übergib `--no-analytics` oder setze `DOZZLE_NO_ANALYTICS=true`. Dann werden keine Beacon-Anfragen gestellt.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
