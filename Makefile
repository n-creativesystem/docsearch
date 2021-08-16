
.PHONY: protoc
protoc:
	@protoc -I./protobuf --go-grpc_out=./protobuf --go-grpc_opt=paths=source_relative --go_out=./protobuf --go_opt=paths=source_relative ./protobuf/*.proto
	@protoc -I./protobuf --grpc-gateway_out ./protobuf --grpc-gateway_opt logtostderr=true,allow_delete_body=true --grpc-gateway_opt paths=source_relative ./protobuf/*.proto