ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder

WORKDIR /usr/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .


FROM debian:bookworm-slim AS geodb

ARG DBIP_MONTH

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl gzip && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    month="${DBIP_MONTH:-$(date -u +%Y-%m)}"; \
    prev_month=$(date -u -d "${month}-01 -1 day" +%Y-%m); \
    url="https://download.db-ip.com/free/dbip-city-lite-${month}.mmdb.gz"; \
    if ! curl -fsSL -o /dbip.mmdb.gz "$url"; then \
        url="https://download.db-ip.com/free/dbip-city-lite-${prev_month}.mmdb.gz"; \
        curl -fsSL -o /dbip.mmdb.gz "$url"; \
    fi; \
    gunzip /dbip.mmdb.gz


FROM debian:bookworm

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /run-app /usr/local/bin/
COPY --from=geodb /dbip.mmdb /data/dbip-city-lite.mmdb
CMD ["run-app"]
