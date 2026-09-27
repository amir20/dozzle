---
title: Anonyme Analysedaten
sourceHash: 8421075e674a
---

# Erhebung von Analysedaten

Dozzle erhebt über einen schlanken Beacon anonyme Nutzungsdaten, um Funktionen und Fehlerbehebungen besser priorisieren zu können. Es ist ein Open-Source-Projekt ohne Finanzierung, deshalb sind diese Daten das wichtigste Signal dafür, wo sich Aufwand lohnt.

## Was wird erhoben

Dozzle sendet beim Start einen Beacon und einen weiteren, wenn jemand die Oberfläche öffnet. Zusammen enthalten sie:

- die Dozzle-Version, den Betriebsmodus (server, swarm, k8s, agent) und die Version der Docker Engine
- welcher Auth-Provider aktiv ist und ob Aktionen und Shell eingeschaltet sind
- kleine Zählwerte: Hosts, Agents, laufende Container und Filter
- den User-Agent-String des Browsers, nur im Beacon der Oberfläche
- die ID der Docker Engine, damit dieselbe Installation nicht doppelt gezählt wird

Es werden niemals Log-Inhalte, Containernamen, Image-Namen, Hostnamen oder Benutzerkennungen übertragen. Die genauen Felder ändern sich mit der Zeit. Maßgeblich ist [`types/beacon.go`](https://github.com/amir20/dozzle/blob/master/types/beacon.go), gesendet wird von [`internal/analytics/http_beacon.go`](https://github.com/amir20/dozzle/blob/master/internal/analytics/http_beacon.go).

## Wo werden die Daten gespeichert

Beacons gehen an `https://b.dozzle.dev/event` und werden von [drain](https://github.com/amir20/drain) empfangen, einem Open-Source-Dienst in Go, der sie für die Auswertung in eine Datenbank und in Parquet-Dateien schreibt. drain speichert nicht die IP-Adresse, von der ein Beacon kam, und die Daten werden an keine Dritten weitergegeben.

## Deaktivieren

Übergib `--no-analytics` oder setze `DOZZLE_NO_ANALYTICS=true`. Dann werden keine Beacon-Anfragen gestellt.

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      DOZZLE_NO_ANALYTICS: "true"
```
