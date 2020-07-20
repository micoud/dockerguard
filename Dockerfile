FROM golang:1.14-alpine as builder
RUN apk add --no-cache ca-certificates git
ENV GO114MODULE=on
WORKDIR /go/src/github.com/micoud/dockerguard
ADD go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/dockerguard ./cmd/dockerguard

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/dockerguard /go/bin/dockerguard

# minimal config might be overriden by mounting another config to /routes.json
COPY routes.json /routes.json

# might be overriden by another command
CMD [ "/go/bin/dockerguard" ]
