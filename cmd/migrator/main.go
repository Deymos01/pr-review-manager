package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Deymos01/pr-review-manager/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Please provide a migration direction: 'up' of 'down'")
	}

	direction := os.Args[1]
	cfg := config.Load()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.PostgresConfig.User,
		cfg.PostgresConfig.Password,
		cfg.PostgresConfig.Host,
		cfg.PostgresConfig.Port,
		cfg.PostgresConfig.DBName,
		cfg.PostgresConfig.SSLMode)

	m, err := migrate.New(cfg.MigrationsPath, dsn)
	if err != nil {
		log.Fatal(err)
	}

	switch direction {
	case "up":
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				log.Println("no new migrations to apply")
				return
			}
			log.Fatal(err)
		}
		log.Println("Migrations applied successfully.")
	case "down":
		if err := m.Down(); err != nil {
			log.Fatal(err)
		}
		log.Println("Migrations rolled back successfully.")
	default:
		log.Fatal("Invalid direction. Use 'up' or 'down'.")
	}
}
