package config

type Config struct {
	MQTTBroker  string
	ServerPort  string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		MQTTBroker:  "tcp://localhost:1883",
		ServerPort:  ":3000",
		DatabaseURL: "postgres://tmsoft:12@127.0.0.1:5432/server"}
}
