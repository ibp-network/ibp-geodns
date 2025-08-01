package data2

import (
	"database/sql"
	"fmt"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	_ "github.com/go-sql-driver/mysql"
)

var DB *sql.DB

func Init() {
	c := cfg.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		c.Local.Mysql.User,
		c.Local.Mysql.Pass,
		c.Local.Mysql.Host,
		c.Local.Mysql.Port,
		c.Local.Mysql.DB,
	)

	var err error
	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Log(log.Fatal, "[data2] MySQL DSN open error: %v", err)
	}

	DB.SetConnMaxIdleTime(2 * time.Minute)
	DB.SetMaxIdleConns(5)
	DB.SetMaxOpenConns(40)
	DB.SetConnMaxLifetime(4 * time.Hour)

	// retry loop (30 s max)
	for i := 0; i < 30; i++ {
		if err = DB.Ping(); err == nil {
			log.Log(log.Info, "[data2] Connected to MySQL (%s)", c.Local.Mysql.Host)
			return
		}
		log.Log(log.Warn, "[data2] MySQL ping failed (%v) — retry %d/30", err, i+1)
		time.Sleep(time.Second)
	}

	log.Log(log.Fatal, "[data2] Unable to connect to MySQL after 30 s: %v", err)
}
