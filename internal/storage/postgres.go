package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	db, err := tryConnect(dsn, 3, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("database not reachable: %w", err)
	}

	// Create table if not exists (best effort)
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

// tryConnect attempts to connect with retries and fallback to localhost if host is postgres.
func tryConnect(dsn string, retries int, timeout time.Duration) (*sql.DB, error) {
	var lastErr error
	for i := 0; i < retries; i++ {
		// Try original DSN
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			lastErr = err
			time.Sleep(1 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			return db, nil
		}
		db.Close()
		lastErr = err

		// If error is hostname resolution and DSN uses "postgres", try localhost
		if strings.Contains(err.Error(), "hostname resolving error") && strings.Contains(dsn, "@postgres:") {
			newDSN := strings.Replace(dsn, "@postgres:", "@localhost:", 1)
			logger.Logger.Info().Msgf("Trying fallback DSN: %s", newDSN)
			db, err = sql.Open("pgx", newDSN)
			if err != nil {
				lastErr = err
				time.Sleep(1 * time.Second)
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err = db.PingContext(ctx)
			cancel()
			if err == nil {
				return db, nil
			}
			db.Close()
			lastErr = err
		}

		time.Sleep(1 * time.Second)
	}
	return nil, lastErr
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
			continue
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
