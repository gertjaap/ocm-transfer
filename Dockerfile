FROM golang:alpine as build-env
RUN apk --no-cache add git
RUN go get github.com/btcsuite/btcd/rpcclient
RUN go get github.com/btcsuite/btcd/chaincfg/chainhash
RUN go get github.com/btcsuite/btcd/wire
RUN go get github.com/gorilla/mux
RUN go get github.com/paulbellamy/ratecounter
RUN mkdir -p /go/src/github.com/gertjaap/ocm-transfer
ADD . /go/src/github.com/gertjaap/ocm-transfer
WORKDIR /go/src/github.com/gertjaap/ocm-transfer
RUN go get ./...
RUN go build -o ocm-transfer

# final stage
FROM alpine
RUN apk --no-cache add ca-certificates libzmq
WORKDIR /app
COPY --from=build-env /go/src/github.com/gertjaap/ocm-transfer/ocm-transfer /app/
ENTRYPOINT ./ocm-transfer