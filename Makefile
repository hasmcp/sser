push-build:
	docker buildx build --platform linux/amd64,linux/arm64 --tag mustafaturan/sser:latest -f Dockerfile --push .
