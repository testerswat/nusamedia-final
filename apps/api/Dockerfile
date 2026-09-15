FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go build -o /out/nusamedia-api ./cmd/server
FROM alpine:3.22
COPY --from=build /out/nusamedia-api /nusamedia-api
EXPOSE 8080
ENTRYPOINT ["/nusamedia-api"]
