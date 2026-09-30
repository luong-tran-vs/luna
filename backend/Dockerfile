FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

# Audio directory owned by the distroless "nonroot" user (uid 65532). A new named volume
# mounted here is initialised from the image, so it keeps this ownership.
RUN mkdir -p /out/data/audio

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /api /api
COPY --from=build --chown=65532:65532 /out/data /data
EXPOSE 8080
ENTRYPOINT ["/api"]
