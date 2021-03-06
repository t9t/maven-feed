FROM golang:1.14 AS build
ARG COMMIT_HASH
ARG BUILD_TIME
WORKDIR /sources
COPY . /sources
RUN go test -count=1 -race ./...
RUN CGOENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./maven-feed.bin -ldflags "-X 'main.commitHash=$COMMIT_HASH' -X 'main.buildTime=$BUILD_TIME'" -tags netgo ./...


FROM scratch

CMD ["/maven-feed"]
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /sources/maven-feed.bin /maven-feed
