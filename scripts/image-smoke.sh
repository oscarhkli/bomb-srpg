#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

docker info >/dev/null 2>&1 || { echo "docker daemon not running"; exit 2; }

tag="bomb-srpg-smoke:plain-$$"
name="bomb-srpg-smoke-plain-$$"

cleanup() {
	status=$?
	trap - EXIT
	docker rm -f "$name" >/dev/null 2>&1 || true
	docker rmi -f "$tag" >/dev/null 2>&1 || true
	exit "$status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

docker build -q -t "$tag" . >/dev/null

port=$(node -e "const s=require('net').createServer().listen(0,()=>{console.log(s.address().port);s.close()})")
docker run -d --name "$name" -p "127.0.0.1:$port:8080" "$tag" >/dev/null

for _ in $(seq 20); do
	docker exec "$name" ./srpg-web healthcheck >/dev/null 2>&1 && break
	sleep 0.5
done
docker exec "$name" ./srpg-web healthcheck

user=$(docker inspect -f '{{.Config.User}}' "$tag")
[ "$user" = "nonroot:nonroot" ] || { echo "image user is '$user', want nonroot:nonroot"; exit 1; }

body=$(curl -fsS "http://127.0.0.1:$port/")
case $body in *"<script"*) ;; *) echo "GET / has no <script"; exit 1 ;; esac

echo smoke-ok
