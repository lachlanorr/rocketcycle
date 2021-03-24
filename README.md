# For Brock... and probably no other human being:

## Build everything - Will make the GNUmakefile
```
$> make
```

## Docker build the rocketcycle-kafka image
```
$> cd docker/images
$> ./build.sh kafka
```

## Run the docker compose stuff to get the infrastructure up
```
$> cd docker/compose/kafka_pg
$> docker-compose up
```

## Create the database schema
```
$> ./examples/rpg/db/init.sh
```

## Update the platform config
```
$> cd build/bin
$> ./rpg platform update
```

## Run all the server stuff... lots of stuff
```
$> cd build/bin
$> ./rpg all
```

## Run the edge client to create a player
```
$> cd build/bin
$> ./rpg edge create player username=brock01
```
