#!/bin/sh
set -eu

curl -sS -X POST "http://localhost:8080/delivery/decide" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"listener-42","asset_id":"asset-8","job_id":"transcode-9","stage":"ready","creator_id":"creator-3"}'
