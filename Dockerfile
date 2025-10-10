FROM golang:1.25.2-alpine3.22 AS scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION=v3.4.0
COPY . /scc
WORKDIR /scc
RUN go build -ldflags="-s -w" -o /bin/scc

FROM alpine:3.22
COPY --from=scc-get /bin/scc /bin/scc
CMD ["/bin/scc"]
