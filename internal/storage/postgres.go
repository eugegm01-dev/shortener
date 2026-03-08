package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/eugegm01-dev/shortener/internal/models"
	"github.com/eugegm01-dev/shortener/pkg/logger"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Retry connection with exponential backoff (critical for CI environments)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		if err = db.PingContext(ctx); err == nil {
			logger.Logger.Info().Msg("Successfully connected to PostgreSQL")
			break
		}
		lastErr = err
		logger.Logger.Warn().Err(err).Int("attempt", attempt).Msg("Waiting for PostgreSQL...")

		// Exponential backoff: 1s, 2s, 3s, ... up to ~15s total
		select {
		case <-time.After(time.Duration(attempt) * time.Second):
			// continue to next attempt
		case <-ctx.Done():
			db.Close()
			return nil, fmt.Errorf("timeout waiting for database: %w", ctx.Err())
		}
	}

	if lastErr != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database after retries: %w", lastErr)
	}

	// Create table (non-blocking - errors logged but don't fail startup)
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS urls (
		id VARCHAR(255) PRIMARY KEY,
		original_url TEXT NOT NULL
	);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to create urls table, continuing")
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Save(url string) (string, error) {
	if url == "" {
		return "", errEmptyURL
	}
	for i := 0; i < 5; i++ {
		id := generateShortID()
		_, err := p.db.Exec("INSERT INTO urls (id, original_url) VALUES ($1, $2)", id, url)
		if err == nil {
			return id, nil
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue // collision, retry
		}
		return "", err
	}
	return "", fmt.Errorf("failed to save after multiple attempts")
}

func (p *PostgresStorage) Get(id string) (string, error) {
	var originalURL string
	err := p.db.QueryRow("SELECT original_url FROM urls WHERE id = $1", id).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errNotFound
		}
		return "", err
	}
	return originalURL, nil
}

func (p *PostgresStorage) GetAll() ([]models.URL, error) {
	rows, err := p.db.Query("SELECT id, original_url FROM urls")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []models.URL
	for rows.Next() {
		var u models.URL
		if err := rows.Scan(&u.ID, &u.URL); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func (p *PostgresStorage) Ping() error {
	return p.db.Ping()
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
