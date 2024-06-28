package resolve

type ResolverConfig struct {
	Nameservers []string `koanf:"nameservers"`
}
