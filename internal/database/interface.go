package database

import "go_plata_task_v2/internal/models"

// DatabaseInterface определяет интерфейс для работы с базой данных
type DatabaseInterface interface {
	CreateQuoteRequest(from, to string) (*models.QuoteRequest, error)
	CreateOrGetPendingQuoteRequest(from, to string) (*models.QuoteRequest, error)
	GetQuoteRequest(id string) (*models.QuoteRequest, error)
	GetPendingQuoteRequestByPair(from, to string) (*models.QuoteRequest, error)
	GetQuote(from, to string) (*models.Quote, error)
	UpdateQuoteRequestStatus(id, status string) error
	UpsertQuote(from, to string, rate float64) error
	GetPendingQuoteRequests() ([]*models.QuoteRequest, error)
	Close() error
}

// Убеждаемся, что DB реализует DatabaseInterface
var _ DatabaseInterface = (*DB)(nil)
