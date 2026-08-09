package storage

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
)


func NewDataBase(connString string) (*sql.DB, error){
	db, err := sql.Open("pgx", connString)

	if err != nil {
		return nil, err
	}

	is_connected := db.Ping()

	if is_connected != nil {
		return nil, is_connected
	}


}