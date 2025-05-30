package models

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type BankAccount struct {
	ID      string  `gorm:"primaryKey"`
	Balance float64 `gorm:"type:decimal(10,2)"`
}

type TransactionLog struct {
	ID            string    `gorm:"primaryKey"`
	FromAccountID string    `gorm:"index"`
	ToAccountID   string    `gorm:"index"`
	Amount        float64   `gorm:"type:decimal(10,2)"`
	Timestamp     time.Time `gorm:"autoCreateTime"`
}

type TransferRequest struct {
	FromAccountID string  `json:"from_account_id"`
	ToAccountID   string  `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type TransferResult struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type DB struct {
	*gorm.DB
}

func NewDB(dsn string) (*DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.AutoMigrate(&BankAccount{}, &TransactionLog{}); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}
	return &DB{db}, nil
}

func MigrateSQLiteToPostgres(sqliteDSN, postgresDSN string) error {
	sqliteDB, err := gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open SQLite DB: %w", err)
	}
	postgresDB, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to open Postgres DB: %w", err)
	}
	if err := postgresDB.AutoMigrate(&BankAccount{}, &TransactionLog{}); err != nil {
		return fmt.Errorf("failed to migrate Postgres schema: %w", err)
	}
	var accounts []BankAccount
	if err := sqliteDB.Find(&accounts).Error; err != nil {
		return fmt.Errorf("failed to read SQLite accounts: %w", err)
	}
	for _, acc := range accounts {
		if err := postgresDB.Create(&acc).Error; err != nil {
			return fmt.Errorf("failed to migrate account %s: %w", acc.ID, err)
		}
	}
	var logs []TransactionLog
	if err := sqliteDB.Find(&logs).Error; err != nil {
		return fmt.Errorf("failed to read SQLite logs: %w", err)
	}
	for _, log := range logs {
		if err := postgresDB.Create(&log).Error; err != nil {
			return fmt.Errorf("failed to migrate log %s: %w", log.ID, err)
		}
	}
	return nil
}
