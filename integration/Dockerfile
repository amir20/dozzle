FROM amir20/docker-alpine-puppeteer:edge

COPY --chown=pptruser:pptruser package*.json /app/
RUN npm ci

COPY --chown=pptruser:pptruser . /app/

CMD [ "npm", "test"]
