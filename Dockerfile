FROM golang:1.26 AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o readmebuilder .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /build/readmebuilder /usr/local/bin/readmebuilder

ENTRYPOINT ["readmebuilder"]
