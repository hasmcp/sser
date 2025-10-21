FROM golang:1.25 AS build

WORKDIR /app
RUN apt-get update && \
  apt-get install -y gcc && \
  rm -rf /var/lib/apt/lists/*

# RUN apt-get install -y ca-certificates
COPY go.mod go.sum ./
RUN GO111MODULE=on go mod download

COPY internal internal
COPY cmd cmd

RUN CGO_ENABLED=0 go build -v -o sser cmd/api-server/main.go

# Create a "nobody" non-root user for the next image by crafting an /etc/passwd
# file that the next image can copy in. This is necessary since the next image
# is based on scratch, which doesn't have adduser, cat, echo, or even sh.
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd

# No need extra files
FROM scratch

EXPOSE 80 443 8889
COPY --from=build /etc/ssl/certs/ /etc/ssl/certs/
COPY --from=build /app/sser /
COPY --from=build /etc_passwd /etc/passwd
COPY --from=build /app/cmd/api-server/public /public
COPY --from=build /app/cmd/api-server/_config /_config

USER nobody

CMD ["/sser"]
