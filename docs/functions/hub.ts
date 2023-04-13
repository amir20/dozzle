import { Handler, HandlerEvent, HandlerContext } from "@netlify/functions";

const handler: Handler = async (event: HandlerEvent, context: HandlerContext) => {
  const response = await fetch("https://hub.docker.com/v2/repositories/amir20/dozzle");
  const data = await response.json();

  return {
    statusCode: 200,
    body: JSON.stringify(data),
  };
};

export { handler };
