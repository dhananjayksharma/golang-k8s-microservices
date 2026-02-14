package config

import "os"

type Config struct {
	Port     string
	MongoURI string
	MongoDB  string
	MongoCol string
}

func Load() Config {
	return Config{
		Port:     getenv("PORT", "8119"), // 127.0.0.1:27017
		MongoURI: getenv("MONGO_URI", "mongodb://user:password123@127.0.0.1:27017"),
		MongoDB:  getenv("MONGO_DB", "message_db"),
		MongoCol: getenv("MONGO_COLLECTION", "orders"),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
