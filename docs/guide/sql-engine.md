---
title: SQL Engine
---

# SQL Engine <Badge type="warning" text="beta" /> <Badge type="tip" text="v8.5x" />

The SQL Engine is a powerful tool that allows you to run SQL queries against your data. It is designed to provide a seamless experience for users who are familiar with SQL and want to interact with their data using a familiar language.

This feature is currently in beta and is available to all users. If you have any feedback or suggestions, please let us know!

## Getting Started

To get started with the SQL Engine, you will need to have a dataset that you can query. Only JSON logs can be queried using SQL. Dozzle leverages the power of WebAssembly to run SQL queries in the browser, which means that your data never leaves your machine.

To start using the SQL Engine, make sure you have JSON logs and navigate to the dropdown and choose `SQL Analytics`. There is also a keyboard shortcut `^+⇧+f` or `⌘+⇧+f` to quickly open the SQL Engine.

## How Does It Work?

The SQL Engine uses WebAssembly to run SQL queries in the browser with DuckDB. When the SQL Engine is first opened, DuckDB WASM is downloaded and initialized in the browser. This could take a while if you are on a slow connection. The SQL Engine then reads _only_ the JSON logs and creates a virtual table in DuckDB. This allows you to run SQL queries against your data in real-time.

The query that Dozzle runs initially is similar to:

```sql
CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'
```

This query creates a table called `logs` and unnests the JSON logs into rows. You can then run SQL queries against this table to analyze your data.

## Example Queries

Here are some example queries that you can run using the SQL Engine:

### Count the number of logs

```sql
SELECT COUNT(*) FROM logs
```

### Filter logs by a specific field

```sql
SELECT * FROM logs WHERE level = 'error'
```

### Group logs by a specific field

```sql
SELECT level, COUNT(*) FROM logs GROUP BY level
```

## Limitations

WebAssembly has some limitations that you should be aware of when using the SQL Engine:

- The SQL Engine only supports structured data such as JSON
- The SQL Engine is limited to running queries in the browser. This means that you cannot run queries that require access to external resources or databases
- There is a maximum of 4GB of memory that can be used by the SQL Engine. If you run out of memory, you will need to refresh the page to clear the memory
