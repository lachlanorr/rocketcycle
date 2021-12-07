// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

//go:generate sh -c "protoc ../../third_party/github.com/lachlanorr/rocketcycle_proto/*.proto -I ../../third_party/github.com/lachlanorr/rocketcycle_proto -I ../../third_party/proto/googleapis --go_out . --go_opt paths=source_relative"
//go:generate sh -c "protoc ../../third_party/github.com/lachlanorr/rocketcycle_proto/portal.proto -I ../../third_party/github.com/lachlanorr/rocketcycle_proto -I ../../third_party/proto/googleapis -I ../../third_party/proto/grpc-gateway --go-grpc_out . --go-grpc_opt paths=source_relative --grpc-gateway_out . --grpc-gateway_opt logtostderr=true --grpc-gateway_opt paths=source_relative --grpc-gateway_opt generate_unbound_methods=true"
//go:generate sh -c "protoc ../../third_party/github.com/lachlanorr/rocketcycle_proto/portal.proto -I ../../third_party/github.com/lachlanorr/rocketcycle_proto -I ../../third_party/proto/googleapis -I ../../third_party/proto/grpc-gateway --openapiv2_out ../rkcy/static/portal/docs --openapiv2_opt logtostderr=true --openapiv2_opt fqn_for_openapi_name=true"

package rkcypb
