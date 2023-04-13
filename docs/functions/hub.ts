import { Handler, HandlerEvent, HandlerContext } from "@netlify/functions";

let cache;
let cacheTime = 0;

const handler: Handler = async (event: HandlerEvent, context: HandlerContext) => {
  if (cache && Date.now() - cacheTime < 1000 * 60 * 10) {
    const headers = {
      "x-cache": "HIT",
    };
    return {
      headers,
      statusCode: 200,
      body: JSON.stringify(cache),
    };
  }

  const response = await fetch("https://hub.docker.com/v2/repositories/amir20/dozzle");
  const data = await response.json();
  const { full_description, ...rest } = data;
  cache = rest;
  cacheTime = Date.now();

  return {
    statusCode: 200,
    body: JSON.stringify(rest),
  };
};

export { handler };
