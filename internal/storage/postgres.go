package storage

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/eugegm01-dev/shortener/internal/models"
	"github.com/jackc/pgx/v5/pgconn"   // for error code checking
	_ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// create table if not exists
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS urls (
			id VARCHAR(255) PRIMARY KEY,
			original_url TEXT NOT NULL
		);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &PostgresStorage{db: db}, nil
}

func (p *PostgresStorage) Save(url string) (string, error) {
	if url == "" {
		return "", errEmptyURL
	}

	// try up to 5 times in case of ID collision
	for i := 0; i < 5; i++ {
		id := generateShortID()
		_, err := p.db.Exec("INSERT INTO urls (id, original_url) VALUES ($1, $2)", id, url)
		if err == nil {
			return id, nil
		}

		// check for duplicate key violation (PostgreSQL error code 23505)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue // try another ID
		}
		return "", err // other error
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
