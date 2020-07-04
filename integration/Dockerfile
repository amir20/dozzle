FROM amir20/docker-alpine-puppeteer:next

COPY package*.json yarn.lock /app/
RUN yarn

COPY . /app/

CMD ["yarn", "test"]
