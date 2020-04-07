FROM amir20/docker-alpine-puppeteer:edge

COPY --chown=pptruser:pptruser package*.json /app/
RUN yarn 

COPY --chown=pptruser:pptruser . /app/

CMD [ "yarn", "test"]
