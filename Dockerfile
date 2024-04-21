FROM golang:1.21 as build-env

WORKDIR /go/src/app
COPY . /go/src/app

WORKDIR /go/src/app/cmd/reti-algodrop
RUN go get
RUN CGO_ENABLED=0 go build -o /go/bin/reti-algodrop
RUN strip /go/bin/reti-algodrop

FROM gcr.io/distroless/static

COPY --from=build-env /go/bin/reti-algodrop /app/
CMD ["/app/reti-algodrop"]
