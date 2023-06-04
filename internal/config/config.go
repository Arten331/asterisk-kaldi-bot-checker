package config

import (
	"os"
	"strconv"
	"strings"
)

const appName = "bot-checker"

type App struct {
	Name string
	Env  string
}

type Agi struct {
	Host string
	Port int
}

type Ari struct {
	Host     string
	Port     int
	User     string
	Password string
	Original string
	Secure   bool
}

type HTTPService struct {
	Port int
}

type Logger struct {
	Level string
}

type AppConfig struct {
	App          App
	Logger       Logger
	HTTPService  HTTPService
	Agi          Agi
	Kaldi        Kaldi
	Ari          Ari
	QueueService QueueConfig
}

type Kaldi struct {
	Host string
	Port int
}

type QueueConfig struct {
	Kafka  KafkaBroker
	Topics TopicsList
}

type (
	TopicsList struct {
		ClickData ProduceTopicConfig
	}
	KafkaBroker struct {
		Host             string
		Port             int
		BootstrapServers []string
	}
	ProduceTopicConfig struct {
		Name string
	}
)

func Init() (AppConfig, error) {
	var config AppConfig

	// default AppConfig
	config = AppConfig{
		App: App{
			Name: appName,
			Env:  os.Getenv("APP_ENV"),
		},
		Logger: Logger{
			Level: GetEnvAsStr("LOG_LEVEL", "DEBUG"),
		},
		HTTPService: HTTPService{
			Port: GetEnvAsInt("API_PORTHTTP", 8080),
		},
		Agi: Agi{
			Host: GetEnvAsStr("AGI_HOST", ""),
			Port: GetEnvAsInt("AGI_PORT", 8888),
		},
		Kaldi: Kaldi{
			Host: GetEnvAsStr("KALDI_HOST", "localhost"),
			Port: GetEnvAsInt("KALDI_PORT", 2700),
		},
		Ari: Ari{
			Host:     GetEnvAsStr("ARI_HOST", "asterisk.local"),
			Port:     GetEnvAsInt("ARI_PORT", 8089),
			Secure:   GetEnvAsBool("ARI_SECURE", true),
			User:     GetEnvAsStr("ARI_USER", "bot_checker"),
			Password: GetEnvAsStr("ARI_PASS", "bot_checker"),
			Original: GetEnvAsStr("ARI_ORIG", "http://bot-checker.local"),
		},
		QueueService: QueueConfig{
			Kafka: KafkaBroker{
				Host:             GetEnvAsStr("KAFKA_HOST", ""),
				Port:             GetEnvAsInt("KAFKA_PORT", 0),
				BootstrapServers: GetEnvAsStrSlice("KAFKA_BOOTSTRAP_SERVERS", []string{}),
			},
			Topics: TopicsList{
				ClickData: ProduceTopicConfig{
					Name: GetEnvAsStr("KAFKA_TOPIC_CLICK", ""),
				},
			},
		},
	}

	return config, nil
}

func GetEnvAsStr(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func GetEnvAsStrSlice(key string, defaultVal []string) []string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.Split(value, ",")
	}

	return defaultVal
}

func GetEnvAsInt(key string, defaultVal int) int {
	valueStr := GetEnvAsStr(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}

	return defaultVal
}

func GetEnvAsBool(key string, defaultVal bool) bool {
	valStr := GetEnvAsStr(key, "")
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}

	return defaultVal
}
