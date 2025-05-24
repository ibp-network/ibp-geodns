package mysql

import (
	"database/sql"
	"fmt"
	"time"

	cfg "ibp-geodns/src/common/config"

	_ "github.com/go-sql-driver/mysql"
)

func Init() {
	c := cfg.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		c.Local.Mysql.User,
		c.Local.Mysql.Pass,
		c.Local.Mysql.Host,
		c.Local.Mysql.Port,
		c.Local.Mysql.DB,
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("Failed to open MySQL DSN: %v", err))
	}

	// Attempt reconnect loop (example: up to 30 seconds).
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		err = DB.Ping()
		if err == nil {
			break
		}
		fmt.Printf("[mysql.Init] Ping failed: %v (retry %d/%d)\n", err, i+1, maxRetries)
		time.Sleep(time.Second) // wait 1s between tries
	}

	if err != nil {
		// If still not connected after max retries, you can either panic or do something else
		panic(fmt.Sprintf("Failed to connect to MySQL after %d retries: %v", maxRetries, err))
	}

	// Optional: Set your connection pool parameters
	DB.SetMaxOpenConns(100)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(time.Hour)

	fmt.Println("[mysql.Init] Connected successfully to MySQL.")
}
