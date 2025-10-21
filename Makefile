push-build:
	docker buildx build -t mustafaturan/sser:latest --platform linux/amd64 -f Dockerfile.linux --push .
