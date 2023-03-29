FROM cypress/included:12.9.0

RUN npm i -g pnpm

WORKDIR /e2e

COPY pnpm-lock.yaml ./
RUN pnpm fetch

COPY package.json tsconfig.json ./
RUN pnpm install --offline
