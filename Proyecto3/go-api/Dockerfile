FROM golang:1-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM alpine:3
COPY --from=build /out/app /app/app
ENTRYPOINT ["/app/app"]