package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
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

	ps := &PostgresStorage{db: db}
	if err := ps.runMigrations(); err != nil {
		// Логируем ошибку, но не прерываем запуск – таблица может уже существовать
		logger.Logger.Error().Err(err).Msg("Migration failed, but continuing")
	}
	return ps, nil
}

// runMigrations читает и выполняет SQL-файл миграции
func (p *PostgresStorage) runMigrations() error {
	// Путь относительно корня проекта (там, откуда запускается бинарник)
	migrationFile := "migrations/0001_create_urls_table.up.sql"
	content, err := os.ReadFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	if _, err := p.db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}
	return nil
}

func (p *PostgresStorage) Save(url string) (string, error) {
	if url == "" {
		return "", errEmptyURL
	}

	// Таблица уже должна существовать после миграции, поэтому ensureTable больше не нужен
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var lastErr error
	for attempt := 1; attempt <= 20; attempt++ {
		err := p.db.PingContext(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("ping failed after retries: %w", lastErr)
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
