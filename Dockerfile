# ---- Stage 1: build the SvelteKit SPA ----
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci 2>/dev/null || npm install
COPY frontend/ ./
# adapter-static writes the SPA into /app/internal/web/build
RUN npm run build

# ---- Stage 2: build the Go binary with the SPA embedded ----
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Replace the committed placeholder with the freshly built SPA before embed.
COPY --from=frontend /app/internal/web/build ./internal/web/build
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /wm-pickems .

# ---- Stage 3: minimal runtime ----
FROM alpine:3.20

# OCI image metadata. VERSION/REVISION/CREATED are injected by CI
# (docker/build-push-action build-args); they default to dev values for
# local builds. org.opencontainers.image.source links the GHCR package
# back to this repository.
ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED=
LABEL org.opencontainers.image.title="world-cup-pool" \
      org.opencontainers.image.description="World Cup 2026 prediction game" \
      org.opencontainers.image.url="https://github.com/oyvhov/world-cup-pool" \
      org.opencontainers.image.source="https://github.com/oyvhov/world-cup-pool" \
      org.opencontainers.image.base.name="https://github.com/floholz/wm-pickems" \
      org.opencontainers.image.licenses="GPL-3.0-only" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CREATED}"

RUN apk add --no-cache ca-certificates tzdata wget \
	&& adduser -D -u 10001 app \
	&& mkdir -p /pb_data \
	&& chown -R app:app /pb_data
COPY --from=backend /wm-pickems /usr/local/bin/wm-pickems
RUN ln -s /usr/local/bin/wm-pickems /usr/local/bin/world-cup-pool
USER app
EXPOSE 8090
VOLUME ["/pb_data"]
HEALTHCHECK --interval=30s --timeout=4s --start-period=10s \
	CMD wget -qO- http://127.0.0.1:8090/api/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["wm-pickems"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/pb_data"]
