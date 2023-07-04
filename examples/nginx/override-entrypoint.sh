#!/bin/sh

# Register first
curl -X POST -H "Content-Type: application/json" \
    -d "{\"OutsideHost\": \"$OUTSIDE\", \"InsideHost\": \"$HOSTNAME:80\"}" \
    http://${PROXY_HOST}/register

/docker-entrypoint.sh "nginx" "-g" "daemon off;"