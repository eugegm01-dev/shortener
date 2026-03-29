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
    files := []string{
        "migrations/0001_create_urls_table.up.sql",
        "migrations/0002_add_unique_original_url.up.sql",
    }
    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read migration file %s: %w", file, err)
        }
        if _, err := p.db.Exec(string(content)); err != nil {
            // Логируем, но не прерываем (ограничение может уже существовать)
            logger.Logger.Warn().Err(err).Str("file", file).Msg("Migration statement failed")
        }
    }
    return nil
}func (p *PostgresStorage) Save(url string) (string, error) {
    if url == "" {
        return "", errEmptyURL
    }
    id := generateShortID()
    // Пытаемся вставить, при конфликте по unique_original_url – возвращаем существующий id
    var existingID string
    err := p.db.QueryRow(`
        INSERT INTO urls (id, original_url) VALUES ($1, $2)
        ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
        RETURNING id
    `, id, url).Scan(&existingID)
    if err != nil {
        return "", err
    }
    return existingID, nil
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

func (p *PostgresStorage) SaveBatch(urls []string) ([]string, error) {
    if len(urls) == 0 {
        return nil, nil
    }
    tx, err := p.db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    ids := make([]string, 0, len(urls))
    for _, url := range urls {
        if url == "" {
            return nil, errEmptyURL
        }
        id := generateShortID()
        var existingID string
        err = tx.QueryRow(`
            INSERT INTO urls (id, original_url) VALUES ($1, $2)
            ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
            RETURNING id
        `, id, url).Scan(&existingID)
        if err != nil {
            return nil, err
        }
        ids = append(ids, existingID)
    }
    if err = tx.Commit(); err != nil {
        return nil, err
    }
    return ids, nil
}