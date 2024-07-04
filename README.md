# keenctl

keenctl is a utility for dynamically configuring Keenetic routers.

## Features

* [x] Configures static routes on Keenetic routers through SSH remote access.
* [x] Provides the ability to resolve DNS and ASN addresses.
* [x] Filters out private or unspecified addresses.
* [x] Adds the ability to run in serve mode.
* [x] Adds support for customizing nameservers.
* [x] Adds support for TTL in DNS and ASN address resolution.
* [ ] Supports the ssh-agent protocol.
  * [ ] Rework `ssh.InsecureIgnoreHostKey()`
* [ ] Manages configuration via a Telegram bot.

## Configuration

The application uses a configuration file to set up router SSH connection,
DNS resolvers, and routes belongs to network interfaces.

A full example configuration is provided in `keenctl.example.toml`.
Below is explanation for each section:

### SSH Configuration

The `[ssh]` section defines the settings for SSH connections to the Keenetic router.

```toml
[ssh]
# The IP address or hostname of the router
host = "192.168.2.1"
# The port to use for the SSH connection
port = 22
# The username for SSH authentication
user = "admin"
# The password for SSH authentication
password = "admin"
# The maximum number of parallel SSH commands
max_parallel_commands = 2
# The timeout for SSH commands
timeout = "2500ms"
```

### DNS Resolver Configuration

The `[resolver.dns]` section specifies the DNS nameservers used for resolving domain names.

```toml
# List of DNS nameservers
nameservers = ["8.8.4.4", "8.8.8.8"]
```

### Network Interfaces Configuration

The `[[interfaces]]` section defines the network interfaces and their routing configurations.

#### Interface: ISP

The first interface named `ISP` has the following settings:

```toml
# The name of the interface
name = "ISP"
# Whether to clean up routes whose address TTL has expired
cleanup = false
# Default routes settings.
defaults = { filters = ["ipv6", "private", "loopback", "unspecified",], auto = true }
routes = [
    # Route configuration with DNS resolver
    { target = "wttr.in", resolver = "dns" },
    # Route configuration with overrided auto flag
    { target = "ifconfig.me", resolver = "dns", auto = false }
]
```

#### Interface: Wireguard0

The second interface named `Wireguard0` has the following settings:

```toml
[[interfaces]]
# The name of the interface
name = "Wireguard0"
# Whether to clean up routes whose address TTL has expired
cleanup = true
# Default routes settings.
defaults = { filters = ["ipv6", "private", "loopback", "unspecified"], auto = true }
routes = [
    # Routes configuration with IP Subnet target
    { resolver = "addr", target = "10.13.13.0/24" },
    # Routes configuration with DNS resolver and specified auto flag and filters
    { resolver = "dns", target = "example.org", auto = false, filters = ["private"] },
    # Routes configuration with ASN resolver for Cloudflare ASN
    { resolver = "asn", target = "13335" }
]
```

### Explanation of fields

* **name**: The name of the network interface.
* **cleanup**: A boolean indicating whether to clean up routes whose address TTL has expired.
* **defaults**: Default settings applied to the interface.
  * **filters**: Defaults list of filters applied to routes.
  * **auto**: A boolean indicating whether to automatically apply the route when the specified gateway is available.
* **routes**: A list of routes to be added to the interface.
  * **target**: The target destination for the route (can be an IP address, CIDR, or domain).
  * **resolver**: The resolver type used for the target (e.g., "dns", "asn").
  * **auto**: A boolean indicating whether to automatically apply the route when the specified gateway is available.
  * **filters**: Specified list of filters applied to routes.

## Filters

The available filters include:

* **ipv6**: Filters out IPv6 addresses.
* **private**: Filters out private IP addresses, according to RFC 1918 for IPv4 addresses and RFC 4193 for IPv6 addresses. These addresses include:
  * **IPv4**: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
  * **IPv6**: Addresses with the prefix fc00::/7 (Unique Local Unicast)
* **loopback**: Filters out loopback addresses.
* **unspecified**:  Filters out unspecified addresses, which are either the IPv4 address "0.0.0.0" or the IPv6 address "::".
