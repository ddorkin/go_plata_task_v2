package models

import (
	"time"
)

// Поддерживаемые валюты согласно заданию
const (
	USD = "USD"
	EUR = "EUR"
	MXN = "MXN"
)

// Список всех поддерживаемых валют
var SupportedCurrencies = []string{USD, EUR, MXN}

// Проверяем, поддерживается ли валюта
func IsSupportedCurrency(currency string) bool {
	return IsSupportedCurrencyFromList(currency, SupportedCurrencies)
}

// Проверяем, поддерживается ли валюта из заданного списка
func IsSupportedCurrencyFromList(currency string, supportedList []string) bool {
	for _, supported := range supportedList {
		if supported == currency {
			return true
		}
	}
	return false
}

// Запрос на обновление котировки
type QuoteRequest struct {
	ID        string    `json:"id" db:"id"`
	From      string    `json:"from" db:"from_currency"` // Базовая валюта (например, "EUR")
	To        string    `json:"to" db:"to_currency"`     // Котируемая валюта (например, "MXN")
	Status    string    `json:"status" db:"status"`      // pending, processing, completed, failed
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Котировка валютной пары
type Quote struct {
	ID        string    `json:"id" db:"id"`
	From      string    `json:"from" db:"from_currency"` // Базовая валюта
	To        string    `json:"to" db:"to_currency"`     // Котируемая валюта
	Rate      float64   `json:"rate" db:"rate"`          // Курс обмена
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Ответ с котировкой
type QuoteResponse struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Rate      float64   `json:"rate"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Запрос на обновление котировки
type UpdateQuoteRequest struct {
	From string `json:"from" validate:"required"` // Базовая валюта (например, "EUR")
	To   string `json:"to" validate:"required"`   // Котируемая валюта (например, "MXN")
}

// Запрос на получение котировки по ID
type GetQuoteByIDRequest struct {
	ID string `json:"id" validate:"required"`
}

// Запрос на получение последней котировки
type GetLatestQuoteRequest struct {
	From string `json:"from" validate:"required"` // Базовая валюта
	To   string `json:"to" validate:"required"`   // Котируемая валюта
}

// Ответ с ошибкой
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Ответ на запрос обновления котировки
type UpdateQuoteResponse struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Status string `json:"status"`
}

// Ответ от внешнего API
type ExternalAPIResponse struct {
	Success bool               `json:"success"`
	Rates   map[string]float64 `json:"rates"`
	Date    string             `json:"date"`
}
