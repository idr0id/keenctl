package resolve

type ResolverConfig struct {
	Dns DnsConfig `koanf:"dns"`
}

type DnsConfig struct {
	Nameservers []string `koanf:"nameservers"`
}
