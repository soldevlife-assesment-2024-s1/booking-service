package database

import (
	"booking-service/config"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"github.com/uptrace/opentelemetry-go-extra/otelsqlx"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
)

var conn *sqlx.DB

func initConnection(cfg *config.DatabaseConfig) *sqlx.DB {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.DBName)

	var err error
	conn, err = otelsqlx.Connect("postgres", dsn, otelsql.WithAttributes(semconv.DBSystemPostgreSQL), otelsql.WithDBName(cfg.DBName))
	if err != nil {
		panic(err)
	}

	// set connection pool
	conn.SetMaxOpenConns(cfg.MaxOpenConns)
	conn.SetMaxIdleConns(cfg.MaxIdleConns)

	// ping
	err = conn.Ping()
	if err != nil {
		panic(err)
	}

	log.Println("Connected to database")

	return conn

}

func GetConnection(cfg *config.DatabaseConfig) *sqlx.DB {
	if conn == nil {
		initConnection(cfg)
	}
	return conn
}
