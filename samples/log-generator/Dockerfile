FROM golang:1.22-alpine AS build

WORKDIR /src
COPY main.go /src/main.go

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/log-generator /src/main.go

FROM alpine:3.19
RUN adduser -D -g '' appuser
USER appuser

COPY --from=build /out/log-generator /usr/local/bin/log-generator
CMD ["/usr/local/bin/log-generator"]
