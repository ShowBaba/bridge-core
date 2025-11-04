package utils

import (
	"os"
	"path/filepath"
	"strconv"

	log "github.com/showbaba/query-bridge/bridge-core/logger"

	"github.com/joho/godotenv"
)

var (
	_config       *Config
	ConfigFactory = defaultConfig
)

type Config struct {
	Port                    string
	JWTSecretKey            string
	DbHost                  string
	DbPort                  int
	DbUser                  string
	DbPassword              string
	DbName                  string
	AuthServiceSecretKey    string
	MailUsername            string
	MailPassword            string
	RabbitmqServerURL       string
	LogStreamKafkaBrokerUrl string
	EncryptionKey           string
	MongoURI                string
	ServerBaseURL           string
	APIToolKitAPIKey        string
}

func GetConfig() Config {
	if _config == nil {
		_config = ConfigFactory()
	}

	return *_config
}

func defaultConfig() *Config {
	dbPort, _ := strconv.Atoi(os.Getenv("DB_PORT"))
	return &Config{
		Port:                    os.Getenv("PORT"),
		JWTSecretKey:            os.Getenv("JWT_SCECRET"),
		DbHost:                  os.Getenv("DB_HOST"),
		DbPort:                  dbPort,
		DbUser:                  os.Getenv("DB_USER"),
		DbPassword:              os.Getenv("DB_PASSWORD"),
		DbName:                  os.Getenv("DB_NAME"),
		AuthServiceSecretKey:    os.Getenv("AUTH_SERVICE_SECRET_KEY"),
		RabbitmqServerURL:       os.Getenv("RABBITMQ_SERVER_URL"),
		MailUsername:            os.Getenv("MAIL_USERNAME"),
		MailPassword:            os.Getenv("MAIL_PASSWORD"),
		LogStreamKafkaBrokerUrl: os.Getenv("LOG_STREAM_KAFKA_BROKER_URL"),
		EncryptionKey:           os.Getenv("ENCRYPTION_KEY"),
		MongoURI:                os.Getenv("MONGO_URI"),
		ServerBaseURL:           os.Getenv("SERVER_BASE_URL"),
		APIToolKitAPIKey:        os.Getenv("APITOOLKIT_APIKEY"),
	}
}

func init() {
	var (
		dir, _   = os.Getwd()
		basepath = filepath.Join(dir, ".env")
	)
	if err := godotenv.Load(basepath); err != nil {
		log.Error("no .env file found")
		panic(err)
	}
}
