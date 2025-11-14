package tests

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/Deymos01/pr-review-manager/internal/config"

	_ "github.com/lib/pq"
)

var (
	cfg        *config.Config
	db         *sql.DB
	httpClient *http.Client
	baseURL    string
)

func init() {
	var err error

	cfg = config.Load()

	db, err = openTestDB(cfg.PostgresConfig)
	if err != nil {
		panic(fmt.Sprintf("failed to init postgres: %v", err))
	}

	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("failed to ping postgres: %v", err))
	}

	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}

	baseURL = fmt.Sprintf("http://%s:%d", cfg.HTTPServerConfig.Host, cfg.HTTPServerConfig.Port)
}

func openTestDB(dbConfig config.PostgresConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.SSLMode)

	testDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	return testDB, nil
}
