push-build:
	docker buildx build --platform linux/amd64,linux/arm64 --tag hasmcp/sser:latest -f Dockerfile --push .

stats:
	docker stats sser

logs:
	docker logs -f sser

update:
	docker stop sser; docker rm sser; docker image prune -f; docker pull hasmcp/sser:latest; docker run --env-file .env -p 80:80 -p 443:443 --name sser -v ./_config:/_config -v ./_storage:/_storage -d --restart always hasmcp/sser:latest

restart:
	docker stop sser; docker rm sser; docker run --env-file .env -p 80:80 -p 443:443 --name sser -v ./_config:/_config -v ./_storage:/_storage -d --restart always hasmcp/sser:latest
