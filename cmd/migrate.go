// cmd/migrate.go
package main

import (
    "log"

    "github.com/Namanv0509/BankTransfer-Temporal-Workflow/internal/models"
)

func main() {
    if err := models.MigrateSQLiteToPostgres("test.sqlite", "host=localhost user=postgres password=secret dbname=bank port=5433 sslmode=disable"); err != nil {
        log.Fatalf("Migration failed: %v", err)
    }
    log.Println("Migration completed")
}