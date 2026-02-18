package db

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectMySQL(dsn string, capempath string) (*gorm.DB, error) {
	// log.Printf("ConnectMySQL capempath:%s", capempath)
	caCert, err := os.ReadFile(capempath)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caCert); !ok {
		panic("Failed to append CA cert")
	}

	// 2. Create TLS config
	tlsConfig := &tls.Config{
		RootCAs: certPool,
	}

	// Register TLS config
	err = mysqlDriver.RegisterTLSConfig("custom", tlsConfig)
	if err != nil {
		panic(err)
	}

	// 3. MySQL URL (DSN)
	// Format:
	// user:password@tcp(host:3306)/dbname?tls=custom&parseTime=true
	// fmt.Println("dsn:::", dsn)
	// 4. Open GORM connection
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	fmt.Println("✅ connected with CA TLS")
	return db, nil
}
