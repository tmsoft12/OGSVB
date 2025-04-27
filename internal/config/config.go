package config

type Config struct {
	MQTTBroker  string
	ServerPort  string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		MQTTBroker:  "tcp://192.168.5.118:1883",
		ServerPort:  ":3000",
		DatabaseURL: "postgres://tmsoft:12@192.168.5.118:5432/server",
	}
}
