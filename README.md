# For Brock... and probably no other human being:

## Build everything - Will make the GNUmakefile
```
$> make
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
