FROM golang:golang:1.22.3-alpine3.20 as scc-get

ENV GOOS=linux \
GOARCH=amd64 \
CGO_ENABLED=0

ARG VERSION=v3.2.0
COPY . /scc
WORKDIR /scc
RUN go build -ldflags="-s -w" -o /bin/scc

FROM alpine:3.20
COPY --from=scc-get /bin/scc /bin/scc
CMD ["/bin/scc"]
