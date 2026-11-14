FROM golang:1.23-alpine AS build
WORKDIR /src
# go.sum is optional: if it isn't committed yet, go mod tidy resolves it here.
COPY go.mod go.sum* ./
COPY main.go ./
RUN go mod tidy && CGO_ENABLED=0 go build -o /out/opcua-connector .

FROM alpine:3.20
COPY --from=build /out/opcua-connector /usr/local/bin/opcua-connector
ENTRYPOINT ["/usr/local/bin/opcua-connector"]
