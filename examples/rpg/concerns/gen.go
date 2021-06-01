//go:generate sh -c "protoc *.proto -I . -I ../../.. -I ../../../third_party/proto/googleapis -I ../../../third_party/proto/grpc-gateway --go_out . --go_opt paths=source_relative --rkcy_out ."

package concerns
