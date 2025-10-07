package database

import (
	"database/sql"
	"fmt"
	"time"

	"go_plata_task_v2/internal/config"
	"go_plata_task_v2/internal/models"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Соединение с базой данных
type DB struct {
	conn   *sql.DB
	logger *logrus.Logger
}

// Создаём новое соединение с базой данных
func New(cfg *config.DatabaseConfig, logger *logrus.Logger) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Проверяем соединение
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn:   conn,
		logger: logger,
	}

	// Создаем таблицы
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return db, nil
}

// Закрываем соединение с базой данных
func (db *DB) Close() error {
	return db.conn.Close()
}

// Создаём необходимые таблицы
func (db *DB) createTables() error {
	// Сначала создаем таблицы
	tableQueries := []string{
		`CREATE TABLE IF NOT EXISTS quote_requests (
			id VARCHAR(36) PRIMARY KEY,
			from_currency VARCHAR(10) NOT NULL,
			to_currency VARCHAR(10) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS quotes (
			id VARCHAR(36) PRIMARY KEY,
			from_currency VARCHAR(10) NOT NULL,
			to_currency VARCHAR(10) NOT NULL,
			rate DECIMAL(20,8) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(from_currency, to_currency)
		)`,
	}

	// Создаем таблицы
	for _, query := range tableQueries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute table query %s: %w", query, err)
		}
	}

	// Создаем индексы для оптимизации и идемпотентности
	indexQueries := []string{
		`CREATE INDEX IF NOT EXISTS idx_quote_requests_status ON quote_requests(status)`,
		`CREATE INDEX IF NOT EXISTS idx_quotes_currencies ON quotes(from_currency, to_currency)`,
		`CREATE INDEX IF NOT EXISTS idx_quote_requests_currencies ON quote_requests(from_currency, to_currency)`,
		// Уникальный индекс для предотвращения дублирования pending запросов
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_pending_quote_requests 
		 ON quote_requests (from_currency, to_currency) 
		 WHERE status = 'pending'`,
	}

	for _, query := range indexQueries {
		if _, err := db.conn.Exec(query); err != nil {
			return fmt.Errorf("failed to execute index query %s: %w", query, err)
		}
	}

	return nil
}

// Создаём новый запрос на обновление котировки
func (db *DB) CreateQuoteRequest(from, to string) (*models.QuoteRequest, error) {
	query := `INSERT INTO quote_requests (id, from_currency, to_currency, status, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, from_currency, to_currency, status, created_at, updated_at`

	now := time.Now()
	request := &models.QuoteRequest{
		ID:        generateID(),
		From:      from,
		To:        to,
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := db.conn.QueryRow(query, request.ID, request.From, request.To, request.Status, request.CreatedAt, request.UpdatedAt).
		Scan(&request.ID, &request.From, &request.To, &request.Status, &request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create quote request: %w", err)
	}

	return request, nil
}

// Обновляем статус запроса на обновление котировки
func (db *DB) UpdateQuoteRequestStatus(id, status string) error {
	query := `UPDATE quote_requests SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := db.conn.Exec(query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update quote request status: %w", err)
	}
	return nil
}

// Получаем запрос на обновление котировки по ID
func (db *DB) GetQuoteRequest(id string) (*models.QuoteRequest, error) {
	query := `SELECT id, from_currency, to_currency, status, created_at, updated_at FROM quote_requests WHERE id = $1`

	request := &models.QuoteRequest{}
	err := db.conn.QueryRow(query, id).Scan(
		&request.ID, &request.From, &request.To, &request.Status, &request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quote request not found")
		}
		return nil, fmt.Errorf("failed to get quote request: %w", err)
	}

	return request, nil
}

// Получаем существующий pending запрос для валютной пары
func (db *DB) GetPendingQuoteRequestByPair(from, to string) (*models.QuoteRequest, error) {
	query := `SELECT id, from_currency, to_currency, status, created_at, updated_at 
			  FROM quote_requests 
			  WHERE from_currency = $1 AND to_currency = $2 AND status = 'pending'`

	request := &models.QuoteRequest{}
	err := db.conn.QueryRow(query, from, to).Scan(
		&request.ID, &request.From, &request.To, &request.Status, &request.CreatedAt, &request.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("pending quote request not found")
		}
		return nil, fmt.Errorf("failed to get pending quote request: %w", err)
	}

	return request, nil
}

// Создаём новый pending запрос или возвращает существующий
func (db *DB) CreateOrGetPendingQuoteRequest(from, to string) (*models.QuoteRequest, error) {
	// Сначала пытаемся найти существующий pending запрос
	existingRequest, err := db.GetPendingQuoteRequestByPair(from, to)
	if err == nil && existingRequest != nil {
		// Обновляем updated_at для существующего запроса
		updateQuery := `UPDATE quote_requests SET updated_at = $1 WHERE id = $2`
		_, err := db.conn.Exec(updateQuery, time.Now(), existingRequest.ID)
		if err != nil {
			db.logger.WithError(err).WithField("request_id", existingRequest.ID).Warn("Failed to update existing request timestamp")
		}
		return existingRequest, nil
	}

	// Если не найден, создаем новый
	return db.CreateQuoteRequest(from, to)
}

// Создаём или обновляем котировку
func (db *DB) UpsertQuote(from, to string, rate float64) error {
	query := `INSERT INTO quotes (id, from_currency, to_currency, rate, created_at, updated_at) 
			  VALUES ($1, $2, $3, $4, $5, $6) 
			  ON CONFLICT (from_currency, to_currency) 
			  DO UPDATE SET rate = $4, updated_at = $6`

	now := time.Now()
	_, err := db.conn.Exec(query, generateID(), from, to, rate, now, now)
	if err != nil {
		return fmt.Errorf("failed to upsert quote: %w", err)
	}

	return nil
}

// Получаем котировку по паре валют
func (db *DB) GetQuote(from, to string) (*models.Quote, error) {
	query := `SELECT id, from_currency, to_currency, rate, created_at, updated_at FROM quotes WHERE from_currency = $1 AND to_currency = $2`

	quote := &models.Quote{}
	err := db.conn.QueryRow(query, from, to).Scan(
		&quote.ID, &quote.From, &quote.To, &quote.Rate, &quote.CreatedAt, &quote.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quote not found")
		}
		return nil, fmt.Errorf("failed to get quote: %w", err)
	}

	return quote, nil
}

// Получаем все ожидающие запросы на обновление котировок
func (db *DB) GetPendingQuoteRequests() ([]*models.QuoteRequest, error) {
	query := `SELECT id, from_currency, to_currency, status, created_at, updated_at FROM quote_requests WHERE status = 'pending' ORDER BY created_at ASC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending quote requests: %w", err)
	}
	defer rows.Close()

	var requests []*models.QuoteRequest
	for rows.Next() {
		request := &models.QuoteRequest{}
		err := rows.Scan(&request.ID, &request.From, &request.To, &request.Status, &request.CreatedAt, &request.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quote request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}

// Генерируем уникальный ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
