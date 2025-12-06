FROM golang:1.22-alpine3.19 AS build
RUN apk add --no-cache git build-base
WORKDIR /src
COPY go.mod .
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o /out/myros

FROM alpine:3.19
RUN apk add --no-cache nftables python3 py3-nfqueue py3-scapy dumb-init openrc
COPY --from=build /out/myros /usr/local/bin/myros
COPY www /www
COPY block_app.py /usr/local/bin/block_app.py
COPY nftables-appblock.conf /etc/nftables-appblock.conf
COPY appblock /etc/init.d/appblock
COPY nfq-helper /etc/init.d/nfq-helper
RUN chmod +x /etc/init.d/appblock /etc/init.d/nfq-helper /usr/local/bin/block_app.py && \
    mkdir -p /var/log && rc-update add appblock default && rc-update add nfq-helper default
EXPOSE 80
ENTRYPOINT ["/usr/bin/dumb-init", "--"]
CMD ["/bin/sh", "-lc", "nft -f /etc/nftables-appblock.conf || true; openrc default; tail -F /var/log/appblock.log /var/log/nfq.log"]
