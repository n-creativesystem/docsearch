FROM golang:1.17.4-alpine as build
ENV TZ=Asia/Tokyo
WORKDIR /src/
COPY protobuf protobuf
COPY go.mod go.sum Makefile ./
RUN apk update && apk add make git
COPY analyzer analyzer
COPY client client
COPY cmd cmd
COPY config config
COPY errors errors
COPY fsm fsm
COPY helper helper
COPY logger logger
COPY marshaler marshaler
COPY metric metric
COPY model model
COPY plugin plugin
COPY registry registry
COPY server server
COPY storage storage
COPY tools tools
COPY utils utils
COPY version version
COPY *.go ./
RUN make build

FROM alpine:3
RUN addgroup -g 70 -S docsearch \
    && adduser -u 70 -S -D -G docsearch -H -h /var/lib/docsearch -s /bin/sh docsearch \
    && mkdir -p /var/lib/docsearch \
    && chown -R docsearch:docsearch /var/lib/docsearch

COPY --from=build --chown=docsearch:docsearch /src/bin/docsearch /usr/local/bin/
WORKDIR /var/lib/docsearch

USER docsearch

EXPOSE 7000
EXPOSE 8000
EXPOSE 9000

ENTRYPOINT [ "docsearch" ]

CMD ["start"]
