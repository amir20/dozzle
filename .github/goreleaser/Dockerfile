FROM goreleaser/goreleaser:latest

RUN go get -u github.com/gobuffalo/packr/packr
RUN apk --no-cache add nodejs-current nodejs-npm && npm i -g npm

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD [""]
