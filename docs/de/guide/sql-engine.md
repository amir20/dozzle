---
title: SQL-Engine
sourceHash: 0387115a7372
---

# SQL-Engine

Die SQL-Engine ist ein mächtiges Werkzeug, mit dem du SQL-Abfragen gegen deine Daten ausführen kannst. Sie richtet sich an alle, die SQL kennen und ihre Daten in einer vertrauten Sprache auswerten wollen.

Diese Funktion ist derzeit in der Beta-Phase und steht allen Benutzern offen. Wenn du Rückmeldungen oder Vorschläge hast, sag uns Bescheid!

## Erste Schritte

Um mit der SQL-Engine zu starten, brauchst du einen Datensatz, den du abfragen kannst. Per SQL lassen sich ausschließlich JSON-Logs abfragen. Dozzle nutzt WebAssembly, um die SQL-Abfragen im Browser auszuführen, deine Daten verlassen deinen Rechner also nie.

Stelle sicher, dass du JSON-Logs hast, öffne dann das Dropdown-Menü und wähle `SQL Analytics`. Es gibt auch das Tastenkürzel `Ctrl+Shift+F` (bzw. `Cmd+Shift+F` unter macOS), um die SQL-Engine schnell zu öffnen.

## Wie funktioniert das?

Die SQL-Engine nutzt WebAssembly, um SQL-Abfragen mit DuckDB im Browser auszuführen. Beim ersten Öffnen wird DuckDB WASM heruntergeladen und im Browser initialisiert. Bei einer langsamen Verbindung kann das eine Weile dauern. Die SQL-Engine liest dann _nur_ die JSON-Logs und legt daraus eine virtuelle Tabelle in DuckDB an. So kannst du deine Daten in Echtzeit per SQL abfragen.

Die Abfrage, die Dozzle zu Beginn ausführt, sieht ungefähr so aus:

```sql
CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'
```

Diese Abfrage erzeugt eine Tabelle namens `logs` und entpackt die JSON-Logs in Zeilen. Anschließend kannst du diese Tabelle per SQL auswerten.

## Beispielabfragen

Hier ein paar Beispiele, die du mit der SQL-Engine ausführen kannst:

### Anzahl der Logs zählen

```sql
SELECT COUNT(*) FROM logs
```

### Logs nach einem bestimmten Feld filtern

```sql
SELECT * FROM logs WHERE level = 'error'
```

### Logs nach einem bestimmten Feld gruppieren

```sql
SELECT level, COUNT(*) FROM logs GROUP BY level
```

### Verschachtelte JSON-Felder abfragen

```sql
SELECT message.path, message.status, message.duration
FROM logs
WHERE message.status >= 400
ORDER BY message.duration DESC
```

### Nach Zeitfenster aggregieren

```sql
SELECT
  date_trunc('minute', timestamp) AS minute,
  COUNT(*) AS error_count
FROM logs
WHERE level = 'error'
GROUP BY minute
ORDER BY minute DESC
```

## Einschränkungen

WebAssembly bringt einige Einschränkungen mit, die du bei der SQL-Engine kennen solltest:

- Die SQL-Engine unterstützt nur strukturierte Daten wie JSON
- Die SQL-Engine führt Abfragen nur im Browser aus. Abfragen, die Zugriff auf externe Ressourcen oder Datenbanken brauchen, sind also nicht möglich
- Die SQL-Engine kann höchstens 4 GB Speicher nutzen. Geht der Speicher aus, musst du die Seite neu laden, um ihn freizugeben
