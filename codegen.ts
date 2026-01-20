import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "graph/schema.graphqls",
  documents: ["assets/**/*.graphql"],
  generates: {
    "assets/types/graphql.ts": {
      plugins: ["typescript", "typescript-operations", "typed-document-node"],
      config: {
        scalars: {
          Time: "string",
          Int64: "number",
          Any: "unknown",
          Map: "Record<string, string>",
        },
      },
    },
  },
};

export default config;
