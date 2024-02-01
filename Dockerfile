FROM golang:1.22rc1-alpine3.19 as scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION=v3.2.0
COPY . /scc
WORKDIR /scc
RUN go build -ldflags="-s -w" -o /bin/scc

FROM alpine:3.19
COPY --from=scc-get /bin/scc /bin/scc
ENTRYPOINT ["/bin/scc"]
