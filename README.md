# Bootstrap Instructions

## Prereqs
The following should be installed as appropriate for your platform:
- [Docker](https://markdownlivepreview.com/)
- [Go](https://golang.org/doc/install) - 1.16.x
- [Protobuf and gRPC compilers with Go support (see versions below)](https://grpc.io/docs/languages/go/quickstart/)
- [gRPC-Gateway plugin](https://github.com/grpc-ecosystem/grpc-gateway#installation)
- [Sqitch with PosgreSQL support](https://sqitch.org/download/)

In order to standardize protoc output, use these versions:
- protoc 3.17.3
- protoc-gen-go 1.26.0
- protoc-gen-go-grpc 1.0.1
- protoc-gen-grpc-gateway 2.0.1
- protoc-gen-openapiv2 2.0.1

## Build everything - Will make the GNUmakefile
```
$ make
```

## Docker build the rocketcycle-kafka image
```
$ cd docker/images && ./build.sh kafka
```

## Run the docker compose stuff to get the infrastructure up
```
$ cd docker/compose/kafka_pg && docker-compose up
```

## Create the database schema
```
$ ./examples/rpg/db/init.sh
```

## Update the platform config
```
$ cd build/bin && ./rpg platform update
```

## Run all the server stuff... lots of stuff
```
$ cd build/bin && ./rpg run
```

## Run the edge client to create a player
```
$ cd build/bin && ./rpg edge create player username=brock01
```
