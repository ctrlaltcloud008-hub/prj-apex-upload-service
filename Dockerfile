FROM golang:1.26.3 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /upload-api ./cmd

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /upload-api /app/upload-api

ENV PORT=8000
EXPOSE 8000

ENTRYPOINT ["/app/upload-api"]
