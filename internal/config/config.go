package config

type Config struct {
	MQTTBroker  string
	ServerPort  string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		MQTTBroker:  "tcp://127.0.0.1:1883",
		ServerPort:  ":3000",
		DatabaseURL: "postgres://admin:secret@127.0.0.1:5432/mydb",
	}
}
