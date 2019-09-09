FROM golang:1.13-alpine

RUN apk --no-cache add git nodejs-current nodejs-npm make g++ bash bzr curl docker rpm && \
  npm i -g npm && \
  wget https://github.com/goreleaser/goreleaser/releases/latest/download/goreleaser_Linux_x86_64.tar.gz && \
  tar -xvzf goreleaser_Linux_arm64.tar.gz -C /usr/local/bin && \
  rm *.gz && \
  go get -u github.com/gobuffalo/packr/packr

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
CMD [""]
