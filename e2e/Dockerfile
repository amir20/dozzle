FROM cypress/included:12.0.2

RUN apt install curl && curl -f https://get.pnpm.io/v6.16.js | node - add --global pnpm

WORKDIR /e2e

COPY pnpm-lock.yaml ./
RUN pnpm fetch

COPY package.json tsconfig.json ./
RUN pnpm install -r --offline
