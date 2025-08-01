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
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC",
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

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		err = DB.Ping()
		if err == nil {
			break
		}
		fmt.Printf("[mysql.Init] Ping failed: %v (retry %d/%d)\n", err, i+1, maxRetries)
		time.Sleep(time.Second)
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MySQL after %d retries: %v", maxRetries, err))
	}

	DB.SetMaxOpenConns(100)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(time.Hour)

	fmt.Println("[mysql.Init] Connected successfully to MySQL.")
}
