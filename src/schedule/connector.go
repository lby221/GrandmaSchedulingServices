package schedule

import (
	"conf"
	"errors"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/thrsafe"
	"strings"
)

var (
	ErrorDatabaseNotSet = errors.New("Database not correctly set up")
)

func getMySQLConnector() (mysql.Conn, error) {
	conn := mysql.New("tcp", "", dbConnection, dbUsername, dbPassword, "schedule")
	err := conn.Connect()

	if err != nil {
		e := initializeMySQLDatabase()
		if e != nil {
			panic(e)
		} else {
			conn = mysql.New("tcp", "", dbConnection, dbUsername, dbPassword, "schedule")
			err = conn.Connect()
		}
	}

	return conn, err
}

func initializeMySQLDatabase() error {
	conn := mysql.New("tcp", "", dbConnection, dbUsername, dbPassword, "test")
	err := conn.Connect()

	if err != nil {
		return ErrorDatabaseNotSet
	}

	// CREATE DATABASE IF NOT CREATED
	stmt_create_db, err := conn.Prepare("CREATE DATABASE IF NOT EXISTS schedule;")

	if err != nil {
		return err
	}

	stmt_create_db.Run()

	conn.Close()

	conn = mysql.New("tcp", "", dbConnection, dbUsername, dbPassword, "schedule")

	err = conn.Connect()

	if err != nil {
		return err
	}

	rows, _, err := conn.Query("SHOW TABLES LIKE '" + strings.Replace(conf.GetGrandmaName(), " ", "_", -1) + "'")
	if err != nil {
		return err
	}

	if len(rows) > 0 {
		return nil
	}

	// CREATE TABLE IN NEW DATABASE
	stmt_create_table, err := conn.Prepare(`CREATE TABLE records_` + strings.Replace(conf.GetGrandmaName(), " ", "_", -1) +
		` ( id INT(6) UNSIGNED AUTO_INCREMENT PRIMARY KEY, service_type TINYINT NOT NULL, endpoint VARCHAR(512) NOT NULL, 
		message_body VARCHAR(512), ttl BIGINT(11) UNSIGNED, sent BOOLEAN DEFAULT TRUE, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP );`)

	if err != nil {
		return err
	}

	stmt_create_table.Run()

	conn.Close()

	return nil
}
