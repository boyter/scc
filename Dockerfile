FROM golang:1.22rc1-alpine3.19 as scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION=v3.2.0
RUN git clone --branch $VERSION --depth 1 https://github.com/boyter/scc
WORKDIR /go/scc
RUN go build -ldflags="-s -w"

FROM alpine:3.19
COPY --from=scc-get /go/scc/scc /bin/
ENTRYPOINT ["scc"]
