# my-ROs

Project untuk App Blocker sederhana menggunakan nftables + NFQUEUE.

## Build & Run (Alpine VM)

- Install dependencies: `apk add go nftables python3 py3-nfqueue py3-scapy openrc`
- Load nftables: `nft -f nftables-appblock.conf`
- Build Go binary: `CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o myros`
- Jalankan services:
  - Copy `appblock` ke `/etc/init.d/appblock`, `nfq-helper` ke `/etc/init.d/nfq-helper`
  - `chmod +x /etc/init.d/*`
  - `rc-update add appblock default && rc-update add nfq-helper default`
  - `rc-service appblock start && rc-service nfq-helper start`

## Docker

- One-line: `docker run -d --privileged -p 80:80 ghcr.io/USERNAME/appblock:latest`
- Compose contoh:
  ```yaml
  services:
    appblock:
      image: ghcr.io/USERNAME/appblock:latest
      privileged: true
      ports:
        - "80:80"
  ```

## WebUI

- Path `www/index.html`. Menampilkan aplikasi (YouTube/TikTok/Spotify) dengan toggle.
- WebSocket `/ws` men-stream JSON: `{ "type": "flow", "txt": "youtube 142.250.1.1:443" }` tiap 2s.
- Reconnect otomatis ketika koneksi putus.

## API

- `POST /api/block` body: `{ "app": "youtube", "on": true }`
- Efek: `nft add/delete element` ke set `block_<app>`.

## Nftables

- File: `nftables-appblock.conf`
- Table `inet myapp`, chain `filter forward`, sets `block_youtube`, `block_tiktok`, `block_spotify`.
- Rule: `tcp dport {80,443} queue num 0` untuk NFQUEUE.
- Load: `nft -f nftables-appblock.conf`.

## OpenRC Services

- `/etc/init.d/appblock` untuk backend Go.
- `/etc/init.d/nfq-helper` untuk helper NFQUEUE.
- Log: `/var/log/appblock.log`, `/var/log/nfq.log`.

## Release

- Tag `v*` akan membuat release otomatis berisi binary untuk `linux/amd64` & `linux/arm64` + checksum.
- Lint: golangci-lint, ruff.
- Docker: multi-arch ke GHCR `ghcr.io/USERNAME/appblock:latest`.

## Download Binary

- Lihat halaman Releases pada GitHub.

## Screenshot WebUI

![UI](./docs/screenshot.png)

## License

MIT
