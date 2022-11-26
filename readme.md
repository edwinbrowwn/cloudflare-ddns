Cloudflare Dynamic DNS

Config

```json
[
  {
    "authEmail": "",
    "authKey": "",
    "zoneIdentifier": "",
    "recordName": "",
    "proxy": true
  }
]
```

Dev

```sh
docker rm cloudflare-ddns-dev || true && docker build -t cloudflare-ddns . && docker run --name cloudflare-ddns-dev -it cloudflare-ddns
```

Deploy

```sh
docker run --name cloudflare-ddns -d cloudflare-ddns
```
