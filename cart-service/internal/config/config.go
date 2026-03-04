package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort int
	MySQL    MySQL
	Kafka    Kafka
}

type MySQL struct {
	User string
	Pass string
	Host string
	Port int
	DB   string
}

type Kafka struct {
	Brokers []string
	Topic   string
	GroupID string
}

func Load() Config {
	return Config{
		HTTPPort: mustInt(getenv("HTTP_PORT", "8915")),
		MySQL: MySQL{
			User: getenv("DB_USER", "root"),
			Pass: getenv("DB_PASS", "root#123PD"),
			Host: getenv("DB_HOST", "127.0.0.1"),
			Port: mustInt(getenv("DB_PORT", "3306")),
			DB:   getenv("DB_NAME", "techies_cart_db"),
		},
		Kafka: Kafka{
			Brokers: strings.Split(getenv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topic:   getenv("KAFKA_TOPIC", "cart.events"),
			GroupID: getenv("KAFKA_GROUP_ID", "cart-service"),
		},
	}
}

func (m MySQL) DSN() string {
	// parseTime=true is required for time.Time mapping
	return m.User + ":" + m.Pass + "@tcp(" + m.Host + ":" + strconv.Itoa(m.Port) + ")/" + m.DB +
		"?parseTime=true&loc=UTC&charset=utf8mb4,utf8&collation=utf8mb4_unicode_ci"
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func mustInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
