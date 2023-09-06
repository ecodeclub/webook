package config

type WebookConfig struct {
	DB DBConfig
}

type DBConfig struct {
	DSN string
}
