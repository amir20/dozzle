import type { Completion, CompletionContext, CompletionSource } from "@codemirror/autocomplete";
import type { StringStream } from "@codemirror/language";
import { createEditorTheme, createHighlightStyle } from "@/composable/editorTheme";

export interface SQLColumn {
  name: string;
  type: string;
}

export interface SQLEditorOptions {
  parent: HTMLElement;
  placeholder: string;
  initialValue: string;
  /** Read lazily: the table only exists once the logs have been loaded into DuckDB. */
  getColumns: () => SQLColumn[];
  /** Bound to Cmd/Ctrl+Enter. */
  onRun?: () => void;
  onChange?: (value: string) => void;
}

const sqlKeywords = [
  "SELECT",
  "FROM",
  "WHERE",
  "GROUP BY",
  "ORDER BY",
  "HAVING",
  "LIMIT",
  "OFFSET",
  "AS",
  "DISTINCT",
  "AND",
  "OR",
  "NOT",
  "IS NULL",
  "IS NOT NULL",
  "IN",
  "LIKE",
  "ILIKE",
  "BETWEEN",
  "CASE",
  "WHEN",
  "THEN",
  "ELSE",
  "END",
  "ASC",
  "DESC",
  "WITH",
  "UNION ALL",
  "JOIN",
  "ON",
  "USING",
];

// DuckDB functions that actually earn their place against log data: counting,
// bucketing by time, and pulling values out of text.
const sqlFunctions: [string, string][] = [
  ["count(*)", "row count"],
  ["count(DISTINCT )", "distinct values"],
  ["sum()", "sum"],
  ["avg()", "average"],
  ["min()", "minimum"],
  ["max()", "maximum"],
  ["median()", "median"],
  ["quantile_cont(, 0.95)", "95th percentile"],
  ["approx_count_distinct()", "cheap distinct count"],
  ["epoch_ms()", "millis to timestamp"],
  ["to_timestamp()", "seconds to timestamp"],
  ["date_trunc('minute', )", "bucket by minute"],
  ["strftime(, '%H:%M')", "format a timestamp"],
  ["regexp_matches(, '')", "regex test"],
  ["regexp_extract(, '')", "regex capture"],
  ["lower()", "lowercase"],
  ["coalesce(, )", "first non-null"],
  ["cast( AS VARCHAR)", "cast a value"],
];

/** DuckDB only accepts a bare identifier for `foo`; `log.level` or `@timestamp` need quoting. */
function quoteColumn(name: string): string {
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(name) ? name : `"${name}"`;
}

function createAutocomplete(getColumns: () => SQLColumn[]): CompletionSource {
  return (context: CompletionContext) => {
    // Quotes and @ are part of a column name here, so a half-typed `"log.` still matches.
    const word = context.matchBefore(/[\w"@.$]+/);
    if (!word && !context.explicit) return null;

    const typed = word ? word.text.replace(/^"/, "").toLowerCase() : "";

    const options: Completion[] = [
      ...getColumns().map((column): Completion => ({
        label: column.name,
        apply: quoteColumn(column.name),
        detail: column.type,
        type: "property",
        boost: 20,
      })),
      { label: "logs", detail: "table", type: "class", boost: 10 },
      ...sqlFunctions.map(([label, detail]): Completion => ({ label, detail, type: "function" })),
      ...sqlKeywords.map((label): Completion => ({ label, type: "keyword" })),
    ];

    const filtered = typed ? options.filter((o) => o.label.toLowerCase().includes(typed)) : options;
    return { from: word ? word.from : context.pos, options: filtered };
  };
}

const keywordSet = new Set(sqlKeywords.flatMap((k) => k.split(" ")));
const typeSet = new Set(["VARCHAR", "BIGINT", "INTEGER", "DOUBLE", "BOOLEAN", "TIMESTAMP", "DATE", "JSON", "STRUCT"]);
const literalSet = new Set(["TRUE", "FALSE", "NULL"]);

/**
 * Minimal SQL tokenizer. Same reasoning as the expr one: without a language attached
 * CodeMirror never assigns highlight tags and the query renders as flat text.
 */
function tokenizeSQL(stream: StringStream): string | null {
  if (stream.eatSpace()) return null;

  if (stream.match("--")) {
    stream.skipToEnd();
    return "comment";
  }

  if (stream.match("/*")) {
    while (!stream.eol()) {
      if (stream.match("*/")) break;
      stream.next();
    }
    return "comment";
  }

  const char = stream.peek();
  if (char === undefined) {
    stream.next();
    return null;
  }

  // Single quotes are string literals; double quotes are quoted identifiers.
  if (char === "'" || char === '"') {
    const quote = stream.next();
    let ch: string | void;
    while ((ch = stream.next()) !== undefined) {
      if (ch === quote) {
        // '' inside a string is an escaped quote, not the end of it.
        if (stream.peek() === quote) {
          stream.next();
          continue;
        }
        break;
      }
    }
    return quote === "'" ? "string" : "propertyName";
  }

  if (stream.match(/^\d+(\.\d+)?/)) return "number";

  if (stream.match(/^[@$]?[A-Za-z_][\w]*/)) {
    const word = stream.current().toUpperCase();
    if (literalSet.has(word)) return "bool";
    if (keywordSet.has(word) || typeSet.has(word)) return "keyword";
    // A word followed by "(" is a call, everything else is a column reference.
    return stream.peek() === "(" ? "function" : "propertyName";
  }

  if (stream.match(/^(<>|!=|>=|<=|\|\||[=<>+\-*/%,;()])/)) return "operator";

  stream.next();
  return null;
}

export async function createSQLEditor(options: SQLEditorOptions) {
  const [
    { EditorView, keymap, placeholder, drawSelection, highlightActiveLine },
    { EditorState, Prec },
    { autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap },
    { HighlightStyle, syntaxHighlighting, StreamLanguage, indentOnInput, bracketMatching },
    { history, historyKeymap, defaultKeymap },
    { tags },
  ] = await Promise.all([
    import("@codemirror/view"),
    import("@codemirror/state"),
    import("@codemirror/autocomplete"),
    import("@codemirror/language"),
    import("@codemirror/commands"),
    import("@lezer/highlight"),
  ]);

  const sqlLanguage = StreamLanguage.define({
    name: "sql",
    token: tokenizeSQL,
    languageData: {
      commentTokens: { line: "--" },
      closeBrackets: { brackets: ["(", "[", "'", '"'] },
    },
  });

  const state = EditorState.create({
    doc: options.initialValue,
    extensions: [
      EditorView.lineWrapping,
      sqlLanguage,
      history(),
      drawSelection(),
      highlightActiveLine(),
      bracketMatching(),
      closeBrackets(),
      indentOnInput(),
      placeholder(options.placeholder),
      autocompletion({
        override: [createAutocomplete(options.getColumns)],
        activateOnTyping: true,
        icons: false,
      }),
      // Above the default keymap so Cmd/Ctrl+Enter runs the query instead of
      // inserting a newline. Plain Enter still breaks the line.
      Prec.high(
        keymap.of([
          {
            key: "Mod-Enter",
            run: () => {
              options.onRun?.();
              return true;
            },
          },
        ]),
      ),
      keymap.of([...closeBracketsKeymap, ...completionKeymap, ...historyKeymap, ...defaultKeymap]),
      createEditorTheme(EditorView, { multiline: true }),
      syntaxHighlighting(createHighlightStyle(HighlightStyle, tags)),
      EditorView.updateListener.of((update) => {
        if (update.docChanged && options.onChange) {
          options.onChange(update.view.state.doc.toString());
        }
      }),
    ],
  });

  return new EditorView({ state, parent: options.parent });
}
