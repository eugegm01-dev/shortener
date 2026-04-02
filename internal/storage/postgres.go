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
        logger.Logger.Error().Err(err).Msg("Migration failed, but continuing")
    }
    return ps, nil
}

// runMigrations читает и выполняет SQL-файл миграции
func (p *PostgresStorage) runMigrations() error {
    files := []string{
        "migrations/0001_create_urls_table.up.sql",
        "migrations/0002_add_unique_original_url.up.sql",
        "migrations/0003_add_user_id_to_urls.up.sql",
        "migrations/0004_add_is_deleted_to_urls.up.sql", // ← добавьте
    }
    for _, file := range files {
        content, err := os.ReadFile(file)
        if err != nil {
            return fmt.Errorf("failed to read migration %s: %w", file, err)
        }
        if _, err := p.db.Exec(string(content)); err != nil {
            logger.Logger.Warn().Err(err).Str("file", file).Msg("Migration failed")
        }
    }
    return nil
}
func (p *PostgresStorage) Save(url string) (string, bool, error) {
    return p.SaveWithUser(url, "") // Без пользователя
}

func (p *PostgresStorage) SaveWithUser(url, userID string) (string, bool, error) {
    if url == "" {
        return "", false, errEmptyURL
    }

    newID := generateShortID()
    var id string

    // Пытаемся вставить новую запись с user_id
    query := `
        INSERT INTO urls (id, original_url, user_id)
        VALUES ($1, $2, $3)
        ON CONFLICT (original_url) DO NOTHING
        RETURNING id
    `

    err := p.db.QueryRow(query, newID, url, userID).Scan(&id)
    if err == sql.ErrNoRows {
        // Конфликт – URL уже существует, получаем существующий id
        err = p.db.QueryRow(
            `SELECT id FROM urls WHERE original_url = $1`,
            url,
        ).Scan(&id)
        if err != nil {
            return "", false, err
        }
        return id, false, nil
    }
    if err != nil {
        return "", false, err
    }
    return id, true, nil
}

func (p *PostgresStorage) Get(id string) (string, error) {
    var originalURL string
    var isDeleted bool
    err := p.db.QueryRow(
        "SELECT original_url, is_deleted FROM urls WHERE id = $1", id,
    ).Scan(&originalURL, &isDeleted)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return "", errNotFound
        }
        return "", err
    }
    if isDeleted {
        return "", errGone
    }
    return originalURL, nil
}
// GetByUser возвращает все ссылки пользователя
func (p *PostgresStorage) GetByUser(userID string) ([]models.UserURL, error) {
    rows, err := p.db.Query(`
        SELECT id, original_url FROM urls
        WHERE user_id = $1 AND is_deleted = false
        ORDER BY id
    `, userID)

    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []models.UserURL
    for rows.Next() {
        var shortID, original string
        if err := rows.Scan(&shortID, &original); err != nil {
            return nil, err
        }
        result = append(result, models.UserURL{
            ShortURL:    "http://localhost:8080/" + shortID, // Базовый URL будет заменен в хендлере
            OriginalURL: original,
        })
    }
    return result, rows.Err()
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
    return p.SaveBatchWithUser(urls, "") // Без пользователя
}

func (p *PostgresStorage) SaveBatchWithUser(urls []string, userID string) ([]string, error) {
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
            INSERT INTO urls (id, original_url, user_id)
            VALUES ($1, $2, $3)
            ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
            RETURNING id
        `, id, url, userID).Scan(&existingID)

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
func (p *PostgresStorage) DeleteUserURLs(userID string, shortIDs []string) error {
    if len(shortIDs) == 0 {
        return nil
    }
    query := `UPDATE urls SET is_deleted = true WHERE id = ANY($1) AND user_id = $2`
    _, err := p.db.Exec(query, shortIDs, userID)
    return err
}
