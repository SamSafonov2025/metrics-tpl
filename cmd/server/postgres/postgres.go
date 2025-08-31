package postgres

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

//postgresql://admin:admin@localhost:5432/admin?schema=public

var Pool *pgxpool.Pool

func Connect(dsn string) {

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}

	log.Println("Successfully connected to database")
}

func InitDB() {
	q := `create table gauge
			(id varchar(256) primary key, 
			value double precision);
		create table counter (
		    id varchar(256) primary key,
		    value integer)`
	_, err := Pool.Exec(context.Background(), q)
	if err != nil {
		log.Fatalf("Unable to create table: %v", err)
	}
}

func Close() {
	Pool.Close()
}
