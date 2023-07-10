#!/bin/bash

cd examples
docker-compose up -d
curl -s http://truck.localhost:2002 | grep -op 'name.*$' | sed -e 's/<[^>]*>//g'
curl -s http://car.localhost:2002 | grep -op 'name.*$' | sed -e 's/<[^>]*>//g'

asciicast2gif -w 80 -h 30 img/demo.cast img/democast.gif