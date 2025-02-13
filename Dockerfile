FROM golang:1.24.0-alpine3.20 AS scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION=v3.4.0
COPY . /scc
WORKDIR /scc
RUN go build -ldflags="-s -w" -o /bin/scc

FROM alpine:3.20
COPY --from=scc-get /bin/scc /bin/scc
CMD ["/bin/scc"]
