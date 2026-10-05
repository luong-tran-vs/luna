FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# Data directory owned by the distroless "nonroot" user (uid 65532); the dictionary is mounted
# under it.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /server /server
COPY --from=build --chown=65532:65532 /out/data /data
EXPOSE 8080
ENTRYPOINT ["/server"]
