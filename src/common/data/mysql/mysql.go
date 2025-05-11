package mysql

import (
	"database/sql"
	"fmt"

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
		panic(fmt.Sprintf("Failed to connect to MySQL: %v", err))
	}

	err = DB.Ping()
	if err != nil {
		panic(fmt.Sprintf("Failed to ping MySQL: %v", err))
	}
}
