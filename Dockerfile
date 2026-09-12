FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/wecratfs ./cmd/res2

FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/wecratfs /usr/local/bin/wecratfs
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/wecratfs"]
