package postgres

import "cosmos"

var _ cosmos.DBService = (*DBService)(nil)

type DBService struct {
	db *DB
}

func NewDBService(db *DB) *DBService {
	return &DBService{
		db: db,
	}
}
