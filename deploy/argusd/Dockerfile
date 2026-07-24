# argusd — the Argus gateway (PREVENT inline + LEARN loop). One static Go binary.
FROM golang:1.26-alpine AS build
WORKDIR /src
# module graph first for layer caching
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/argusd ./cmd/argusd

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/argusd /argusd
EXPOSE 8088
ENTRYPOINT ["/argusd"]
