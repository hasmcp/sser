push-build:
	docker buildx build --platform linux/amd64,linux/arm64 --tag hasmcp/sser:latest -f Dockerfile --push .
