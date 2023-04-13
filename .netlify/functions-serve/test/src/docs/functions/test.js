var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all) __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if ((from && typeof from === "object") || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, {
          get: () => from[key],
          enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable,
        });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// docs/functions/test.ts
var test_exports = {};
__export(test_exports, {
  handler: () => handler,
});
module.exports = __toCommonJS(test_exports);
var handler = async (event, context) => {
  const response = await fetch("https://hub.docker.com/v2/repositories/amir20/dozzle");
  const data = await response.json();
  return {
    statusCode: 200,
    body: JSON.stringify(data),
  };
};
// Annotate the CommonJS export names for ESM import in node:
0 &&
  (module.exports = {
    handler,
  });
//# sourceMappingURL=test.js.map
