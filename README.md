# For Brock... and probably no other human being:

## Build everything - Will make the GNUmakefile
```
$> make
```

## (In a new terminal) Run the docker compose stuff to get the infrastructure up
```
$> cd docker/compose/kafka_pg
$> docker-compose up
```

## Create the database schema
```
$> ./examples/rpg/db/init.sh
```

## (In a new terminal) Run all the server stuff... lots of stuff
```
$> cd build/bin
$> ./rpg all
```

## (In a new terminal) Run the edge client to create a player
```
$> cd build/bin
$> ./rpg edge create player username=brock01
```
