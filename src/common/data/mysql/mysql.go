package mysql

import (
	"database/sql"
	"fmt"

	"ibp-geodns/src/common/config"

	_ "github.com/go-sql-driver/mysql"
)

func Init() {
	c := config.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		c.System.Mysql.User,
		c.System.Mysql.Pass,
		c.System.Mysql.Host,
		c.System.Mysql.Port,
		c.System.Mysql.DB,
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
