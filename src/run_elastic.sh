#!/bin/bash

elastic_name="el"
elastic_password="elastic"

if [[ -n $1 ]]; then
  elastic_name="$1"
fi

if [[ -n $2 ]]; then
  elastic_password="$2"
fi

docker network create elastic 2>/dev/zero || true

output="$(docker run \
  --name "$elastic_name"@ \
  --net elastic \
  -d \
  -p 9200:9200 \
  -p 9300:9300 \
  -e "ELASTIC_PASSWORD=$elastic_password" \
  elasticsearch:8.13.0 \
  >/dev/zero \
  2>/dev/zero)"

status=$?
if [[ $status -ne 0 && $status -ne 125 ]]; then
  echo "$output" 1>&2
  exit $status
fi

if [[ $status -eq 125 ]]; then
  output="$(docker start el)"
  status=$?

  if [[ $status -ne 0 ]]; then
    echo "$output" 1>&2
    exit $status
  fi
fi

echo "Started elasticsearch container" 1>&2
echo "Copying certificate..." 1>&2

docker cp el:/usr/share/elasticsearch/config/certs/http_ca.crt . 2>/dev/zero

status=$?
while [[ $status -ne 0 ]]; do
  sleep 1
  docker cp el:/usr/share/elasticsearch/config/certs/http_ca.crt . 2>/dev/zero
  status=$?
done

echo "Certificate copied!" 1>&2
