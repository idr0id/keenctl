# keenctl

keenctl is a utility for dynamically configuring Keenetic routers.

## Features

* [x] Configures static routes on Keenetic routers through SSH remote access.
* [x] Provides the ability to resolve DNS and ASN addresses.
* [x] Filters out private or unspecified addresses.
* [x] Adds the ability to run in serve mode.
* [x] Adds support for customizing nameservers.
* [ ] Adds support for TTL in DNS and ASN address resolution.
* [ ] Supports the ssh-agent protocol.
  * [ ] Rework `ssh.InsecureIgnoreHostKey()`
* [ ] Allows configuration of the DNS server host.
* [ ] Allows filtering of specific addresses or subnets.
* [ ] Manages configuration via a Telegram bot.

## Configuration

```toml
[ssh]
host = "192.168.2.1"
port = 22
user = "admin"
password = "admin"
max_parallel_commands = 2
timeout = "2500ms"

[[interfaces]]
name = "ISP"
cleanup = false
defaults = { filters = ["ipv6", "private", "loopback", "unspecified"], auto = true }
routes = [
    { target = "wttr.in", resolver = "dns" }
]

[[interfaces]]
name = "Wireguard0"
cleanup = true
defaults = { filters = ["ipv6", "private", "loopback", "unspecified"], auto = true }
routes = [
    { target = "10.13.13.0/24" },
    { resolver = "dns", target = "example.org" },
    # Cloudflare
    { resolver = "asn", target = "13335" },
]

```
