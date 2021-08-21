VERSION ?=
ifeq ($(VERSION),)
  VERSION = latest
endif

NAME	:= docsearch
SRCS	:= $(shell find . -type d -name archive -prune -o -type f -name '*.go')
LDFLAGS	:= -ldflags="-X \"github.com/n-creativesystem/docsearch/version.Version=$(VERSION)\""

build/docker:
	docker build -t ${IMAGE_NAME} .

build: $(SRCS)
	go build $(LDFLAGS) -o bin/$(NAME)
	chmod +x bin/$(NAME)

.PHONY: deps
deps:
	go get -v

.PHONY: cross-build
cross-build: deps
	for os in darwin linux windows; do \
		for arch in amd64 386; do \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o dist/$$os-$$arch/$(NAME); \
		done; \
	done

.PHONY: protoc
protoc:
	@protoc -I./protobuf \
		--go-grpc_out=./protobuf \
		--go-grpc_opt=paths=source_relative \
		--go_out=./protobuf \
		--go_opt=paths=source_relative \
		--validate_out="lang=go:." \
		--grpc-gateway_out ./protobuf \
		--grpc-gateway_opt logtostderr=true,allow_delete_body=true \
		--grpc-gateway_opt paths=source_relative \
		--openapiv2_out ./protobuf \
		--openapiv2_opt logtostderr=true,allow_delete_body=true \
		./protobuf/*.proto


.PHONY: ssl
ssl:
	@openssl req -x509 -nodes -days 3650 -newkey rsa:2048 -keyout ssl/server.key -out ssl/server.crt -subj "/C=JP/ST=Osaka/L=Osaka/O=NCreativeSystem, Inc./CN=localhost"
