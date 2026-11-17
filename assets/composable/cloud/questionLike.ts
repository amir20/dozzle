// Words a typed question opens with. English only, on purpose: this only decides
// whether the palette lists Ask above the container matches, and a question in
// another language still gets the Ask row, just lower down.
const QUESTION_WORDS = new Set([
  "why",
  "what",
  "how",
  "which",
  "who",
  "when",
  "where",
  "is",
  "are",
  "was",
  "were",
  "did",
  "does",
  "do",
  "can",
  "could",
  "should",
  "any",
  "anything",
  "explain",
  "show",
  "summarize",
]);

/**
 * Whether a palette query reads like something to ask rather than something to
 * find. Container names are one word, so a question needs three, or a question
 * mark: "why is postgres slow" is a question, "postgres" and "why" are not.
 */
export function isQuestionLike(query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return false;
  const words = q.split(/\s+/);
  if (q.endsWith("?")) return words.length >= 2;
  return words.length >= 3 && QUESTION_WORDS.has(words[0]);
}
