package resolve

type ResolverConfig struct {
	DNS DNSConfig `koanf:"dns"`
}

type DNSConfig struct {
	Nameservers []string `koanf:"nameservers"`
}
