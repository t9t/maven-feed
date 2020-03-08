FROM golang:1.14 AS build
WORKDIR /sources
COPY . /sources
RUN go test -count=1 -race ./...
RUN CGOENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./maven-feed.bin -tags netgo ./...


FROM scratch

CMD ["/maven-feed"]

COPY --from=build /sources/maven-feed.bin /maven-feed
