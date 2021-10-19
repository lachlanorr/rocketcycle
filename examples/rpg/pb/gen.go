//go:generate sh -c "protoc *.proto -I . -I ../../../third_party/github.com/lachlanorr/rocketcycle_proto --go_out . --go_opt paths=source_relative --rkcy_out ."

package pb
