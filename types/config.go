package types

// HTTPConfig TOML HTTP config section
type HTTPConfig struct {
	ClientAddr string `toml:"client_addr"`
	ClientPort int    `toml:"client_port"`
	ServerAddr string `toml:"server_addr"`
	ServerPort int    `toml:"server_port"`
	Host       string
	CustomIDs  bool `toml:"custom_ids"`
	TLS        HTTPTLSConfig
	Password   string
}

// HTTPTLSConfig TOML HTTP TLS config section
type HTTPTLSConfig struct {
	Enabled bool
	Cert    string
	Key     string
}

// TCPConfig TOML TCP config section
type TCPConfig struct {
	Client   string
	Server   string
	Password string
}

// Config TOML config
type Config struct {
	HTTP HTTPConfig `toml:"http"`
	TCP  TCPConfig  `toml:"tcp"`
}
