# Rocketcycle
Rocketcycle is a stream oriented middle tier application platform. It is written in [Go](https://go.dev) and utilizes [Kafka](https://kafka.apache.org/) extensively.

If you consider a traditional three tier architecture, with storage, logic, and presentation tiers (from bottom to top), Rocketcycle is concerned with the organization of code within the logic tier.

Rocketcycle departs from a traditional logic tier which is characterized by instances of identical application hosts, where requests are made through http and load is balanced with a reverse proxy. In Rocketcycle, the work load is spread across a series of *Concerns*, which encompass the data model and are serviced by dedicated, parititioned Kafka topics. There are no longer any http request/response style messages passed between hosts, but every logical business transaction passes through the Kafka partitions dedicated to each *Concern* the transaction is dealing with.

Once the data model has been decomposed into *Concerns*, multistep transactions can be described and submitted into the stream. The state of a transaction is preserved as it is being processed, since the transaction state at each progressive step is resubmitted in the stream.

The Rocketcycle design leads to *Transaction Oriented Programming*, which is a novel way to achieve something similar to ACID transaction guarantees, without depending on a relational database for these guarantees. Although Rocketcycle transactions are not as powerful as ACID transactions, they do fullfill a form of data model consistency under arbitrarily high amounts of concurrent access to records in the data model.

This approach leads to decreased stress on the relational database, which now primarily serves as the durable layer which also can provide some constraint guarantees when creating new records. Rocketcycle provides caching for all reads, so in a read heavy application the database is frequently never touched. When modifications to the data model happen through the application, these are done asynchronously, but with ordering guarantees, so even in a write heavy situation, application response time is still not dependent upon database response time.

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
