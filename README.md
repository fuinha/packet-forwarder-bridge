# LoRa Gateway Bridge / Packet Forwarder Bridge - The Things Network Fork

LoRa Gateway Bridge (or Packet Forwarder Bridge) is a service that does protocol translation between the
[packet_forwarder UDP protocol](https://github.com/Lora-net/packet_forwarder/blob/master/PROTOCOL.TXT)
running on most LoRa gateways and the Protocol Buffers over gRPC that are used in 
[The Things Network's backend](https://github.com/TheThingsNetwork/ttn).

## Installation

You can download binaries for [macOS][darwin-amd64], [64 bit Linux][linux-amd64], [32 bit Linux][linux-386], [arm Linux][linux-arm], [64 bit Windows][windows-amd64] or [32 bit Windows][windows-386]. You can also install from source with the following steps:

[darwin-amd64]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-darwin-amd64.zip
[linux-amd64]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-linux-amd64.zip
[linux-386]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-linux-386.zip
[linux-arm]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-linux-arm.zip
[windows-amd64]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-windows-amd64.exe.zip
[windows-386]:https://ttnreleases.blob.core.windows.net/gateway-bridge/master/lora-gateway-bridge-windows-386.exe.zip

- Make sure you have [Go](https://golang.org) installed (version 1.7 or later).
- Set up your [Go environment](https://golang.org/doc/code.html#GOPATH)
- Clone our fork: `git clone https://github.com/TheThingsNetwork/packet-forwarder-bridge.git $GOPATH/src/github.com/brocaar/lora-gateway-bridge`
- Go to the folder: `cd $GOPATH/src/github.com/brocaar/lora-gateway-bridge`
- Build: `make deps build install`

## Running

```
lora-gateway-bridge [options]
```

### Options

| **Flag**                 | **ENV Var**            | **Description** |
| ------------------------ | ---------------------- | --------------- |
| `--udp-bind`             | `UDP_BIND`             | ip:port to bind the UDP listener to (default: "0.0.0.0:1700") | 
| `--ttn-account-server`   | `TTN_ACCOUNT_SERVER`   | Account Server URL (default: "https://account.thethingsnetwork.org") |
| `--ttn-account-client-id` | `TTN_ACCOUNT_CLIENT_ID` | Client ID to authenticate with the Account Server (optional) |
| `--ttn-account-client-secret` | `TTN_ACCOUNT_CLIENT_SECRET` | Client Secret to authenticate with the Account Server (optional) |
| `--ttn-discovery-server` | `TTN_DISCOVERY_SERVER` | host:port of TTN Discovery Server | 
| `--root-ca-file`         | `ROOT_CA_FILE`         | Root CA file (if discovery server uses TLS) | 
| `--ttn-router`           | `TTN_ROUTER`           | TTN Router ID |
| `--ttn-gateway-eui`      | `TTN_GATEWAY_EUI`      | TTN Gateway EUI (optional) |
| `--ttn-gateway-id`       | `TTN_GATEWAY_ID`       | TTN Gateway ID the EUI should translate to (optional) |
| `--ttn-gateway-token`    | `TTN_GATEWAY_TOKEN`    | TTN Gateway Token to authenticate the Gateway ID (optional) |
| `--ttn-gateway-key`      | `TTN_GATEWAY_KEY`      | TTN Gateway Key to authenticate the Gateway ID (optional) |

### Example

If you set up the TTN backend for development (as described in the [README](https://github.com/TheThingsNetwork/ttn/#set-up-the-things-networks-backend-for-development)) you can use the following to start the bridge:

```
lora-gateway-bridge --ttn-discovery-server localhost:1900 --ttn-router dev
```

## Issues / Feature Requests

Issues or feature requests for the bridge can be opened at [github.com/brocaar/lora-gateway-bridge/issues](https://github.com/brocaar/lora-gateway-bridge/issues). Issues or feature requests related to The Things Network can be opened at [github.com/TheThingsNetwork/ttn/issues](https://github.com/TheThingsNetwork/ttn/issues).

## License

LoRa Gateway Bridge is distributed under the MIT license. See 
[LICENSE](https://github.com/brocaar/lora-gateway-bridge/blob/master/LICENSE).
