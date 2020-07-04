FROM amir20/docker-alpine-puppeteer:v1

COPY package*.json yarn.lock /app/
RUN yarn

COPY . /app/

CMD ["yarn", "test"]
