//go:generate sh -c "protoc *.proto -I . -I ../../.. -I ../../../third_party/proto/googleapis -I ../../../third_party/proto/grpc-gateway --go_out . --go_opt paths=source_relative"
//go:generate sh -c "protoc edge.proto -I . -I ../../.. -I ../../../third_party/proto/googleapis -I ../../../third_party/proto/grpc-gateway --go-grpc_out . --go-grpc_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative --grpc-gateway_opt generate_unbound_methods=true --openapiv2_out ../rpg_edge/static/docs --openapiv2_opt logtostderr=true --openapiv2_opt fqn_for_openapi_name=true"

package pb
