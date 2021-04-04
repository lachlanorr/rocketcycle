// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:generate sh -c "protoc *.proto -I . -I ../../third_party/proto/googleapis --go_out . --go_opt paths=source_relative"
//go:generate sh -c "protoc admin.proto -I . -I ../../third_party/proto/googleapis -I ../../third_party/proto/grpc-gateway --go-grpc_out . --go-grpc_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative --grpc-gateway_opt generate_unbound_methods=true --openapiv2_out ../../pkg/rkcy/static/admin/docs --openapiv2_opt logtostderr=true --openapiv2_opt fqn_for_openapi_name=true"

package rkcy
