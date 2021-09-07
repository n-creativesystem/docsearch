module github.com/n-creativesystem/docsearch/client

go 1.16

replace github.com/n-creativesystem/docsearch/protobuf => ../protobuf

require (
	github.com/golang/protobuf v1.5.2
	github.com/n-creativesystem/docsearch/protobuf v1.0.0
	google.golang.org/grpc v1.40.0
	google.golang.org/protobuf v1.27.1
)
