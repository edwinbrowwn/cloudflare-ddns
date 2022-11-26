## Cloudflare Dynamic DNS

This application will update a cloudflare dns record with the current public ipv4 address of the machine it is running on. The application will check for changes to the public ipv4 address every 20 seconds and update the dns record if the address has changed. The application will also check for changes to the dns record every 5 minutes and update the address file if the dns record has changed.

#### Configuration

The application requires a config.json file in the same directory as the executable in the following format:

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

#### Dev

```sh
docker rm cloudflare-ddns-dev || true && docker build -t cloudflare-ddns . && docker run --name cloudflare-ddns-dev -it cloudflare-ddns
```

#### Deploy

```sh
docker run --name cloudflare-ddns -d cloudflare-ddns
```
