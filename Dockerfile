FROM golang:1.27.4 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY views/ ./views/
COPY css/ ./css/
COPY static/ ./static/

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/site ./cmd \
    && mkdir -p /out/cache && touch /out/cache/.keep && chown -R 65532:65532 /out/cache

FROM gcr.io/distroless/static-debian12:nonroot AS site

WORKDIR /app

COPY --from=build /out/site ./site
COPY --from=build /out/cache /app/.cache
COPY views/ ./views/
COPY css/ ./css/
COPY static/ ./static/

EXPOSE 8443
USER 65532:65532

CMD ["./site"]
