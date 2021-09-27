module github.com/n-creativesystem/docsearch

go 1.16

require (
	github.com/BBVA/raft-badger v1.1.0
	github.com/armon/go-metrics v0.3.9 // indirect
	github.com/blugelabs/bluge v0.1.7
	github.com/blugelabs/bluge_segment_api v0.2.0
	github.com/dgraph-io/badger/v3 v3.2103.1
	github.com/envoyproxy/protoc-gen-validate v0.6.1
	github.com/fatih/color v1.12.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190910122728-9d188e94fb99 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.5.0
	github.com/hashicorp/go-hclog v0.16.2
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/go-msgpack v1.1.5 // indirect
	github.com/hashicorp/raft v1.3.1
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/ikawaha/blugeplugin v1.3.6
	github.com/ikawaha/kagome-dict v1.0.4
	github.com/ikawaha/kagome-dict/ipa v1.0.4
	github.com/ikawaha/kagome/v2 v2.7.0
	github.com/klauspost/compress v1.13.4 // indirect
	github.com/lyft/protoc-gen-star v0.5.3 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/n-creativesystem/docsearch/client v1.0.0
	github.com/n-creativesystem/docsearch/protobuf v1.0.0
	github.com/natefinch/lumberjack v2.0.0+incompatible
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.0
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d // indirect
	golang.org/x/sys v0.0.0-20210817190340-bfb29a6856f2 // indirect
	golang.org/x/text v0.3.7
	google.golang.org/grpc v1.40.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
)

replace (
	github.com/n-creativesystem/docsearch/client => ./client
	github.com/n-creativesystem/docsearch/protobuf => ./protobuf
)
