---
title: SQL 引擎
sourceHash: 0387115a7372
---

# SQL 引擎

SQL 引擎是一个强大的工具，让你可以对自己的数据运行 SQL 查询。它面向熟悉 SQL、希望用熟悉的语言操作数据的用户，提供顺畅的使用体验。

该功能目前处于 beta 阶段，所有用户都可以使用。如果你有任何反馈或建议，欢迎告诉我们！

## 快速开始

要开始使用 SQL 引擎，你需要有可供查询的数据集。只有 JSON 日志才能用 SQL 查询。Dozzle 借助 WebAssembly 在浏览器中运行 SQL 查询，也就是说你的数据永远不会离开你的机器。

要开始使用 SQL 引擎，请确认你有 JSON 日志，然后打开下拉菜单并选择 `SQL Analytics`。也可以使用快捷键 `Ctrl+Shift+F`（macOS 上是 `Cmd+Shift+F`）快速打开 SQL 引擎。

## 它是如何工作的？

SQL 引擎使用 WebAssembly 在浏览器中通过 DuckDB 运行 SQL 查询。首次打开 SQL 引擎时，DuckDB WASM 会被下载并在浏览器中初始化。如果你的网络较慢，这可能需要一些时间。随后 SQL 引擎_只_读取 JSON 日志，并在 DuckDB 中创建一张虚拟表，从而让你可以实时地对数据运行 SQL 查询。

Dozzle 最初运行的查询类似于：

```sql
CREATE TABLE logs AS SELECT unnest(m) FROM 'logs.json'
```

这个查询会创建一张名为 `logs` 的表，并把 JSON 日志展开成行。之后你就可以对这张表运行 SQL 查询来分析数据。

## 查询示例

下面是一些可以在 SQL 引擎中运行的查询示例：

### 统计日志条数

```sql
SELECT COUNT(*) FROM logs
```

### 按某个字段过滤日志

```sql
SELECT * FROM logs WHERE level = 'error'
```

### 按某个字段分组

```sql
SELECT level, COUNT(*) FROM logs GROUP BY level
```

### 查询嵌套的 JSON 字段

```sql
SELECT message.path, message.status, message.duration
FROM logs
WHERE message.status >= 400
ORDER BY message.duration DESC
```

### 按时间窗口聚合

```sql
SELECT
  date_trunc('minute', timestamp) AS minute,
  COUNT(*) AS error_count
FROM logs
WHERE level = 'error'
GROUP BY minute
ORDER BY minute DESC
```

## 限制

WebAssembly 有一些限制，使用 SQL 引擎时需要注意：

- SQL 引擎只支持 JSON 之类的结构化数据
- SQL 引擎只能在浏览器中运行查询，因此无法运行需要访问外部资源或数据库的查询
- SQL 引擎最多只能使用 4GB 内存。如果内存耗尽，你需要刷新页面来释放内存
