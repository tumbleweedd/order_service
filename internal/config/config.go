package config

import (
	"flag"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string         `yaml:"env" env-default:"local"`
	HTTP     HTTPConfig     `yaml:"http"`
	Postgres PostgresConfig `yaml:"postgres"`
	Kafka    KafkaConfig    `yaml:"kafka"`
}

type HTTPConfig struct {
	Port int `yaml:"port"`
}

type PostgresConfig struct {
	Port    string `yaml:"port"`
	Host    string `yaml:"host"`
	DbName  string `yaml:"db_name"`
	User    string `yaml:"user"`
	Pwd     string `yaml:"password"`
	SslMode string `yaml:"sslmode"`
}

type KafkaConfig struct {
	BrokerList       []string `yaml:"broker_list"`
	OrderEventTopic  string   `yaml:"order_event_topic"`
	StatusEventTopic string   `yaml:"status_event_topic"`
}

func InitConfig() Config {
	configPath := getConfigPath()

	if configPath == "" {
		panic("config path is not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("config file does not exist: " + configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic("empty config path: " + err.Error())
	}

	return cfg
}

func getConfigPath() string {
	var path string

	flag.StringVar(&path, "config", "", "path to config file")
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
	}

	return path
}
