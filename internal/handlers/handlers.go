package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go_plata_task_v2/internal/database"
	"go_plata_task_v2/internal/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

//  Зависимости для обработчиков
type Handler struct {
	db                  database.DatabaseInterface
	logger              *logrus.Logger
	supportedCurrencies []string
}

// Создаём новый экземпляр Handler
func New(db database.DatabaseInterface, logger *logrus.Logger, supportedCurrencies []string) *Handler {
	return &Handler{
		db:                  db,
		logger:              logger,
		supportedCurrencies: supportedCurrencies,
	}
}

// @Summary Обновить котировку валютной пары
// @Description Создает запрос на обновление котировки валютной пары (например, EUR/MXN). Обновление происходит в фоновом режиме.
// @Tags quotes
// @Accept json
// @Produce json
// @Param request body models.UpdateQuoteRequest true "Запрос на обновление котировки"
// @Success 200 {object} models.UpdateQuoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /quotes/update [post]
func (h *Handler) UpdateQuote(w http.ResponseWriter, r *http.Request) {
	var req models.UpdateQuoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err.Error())
		return
	}

	// Валидация
	if strings.TrimSpace(req.From) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "From currency is required")
		return
	}
	if strings.TrimSpace(req.To) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "To currency is required")
		return
	}

	// Нормализуем валюты (делаем заглавными)
	from := strings.ToUpper(strings.TrimSpace(req.From))
	to := strings.ToUpper(strings.TrimSpace(req.To))

	// Проверяем, что валюты разные
	if from == to {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "From and To currencies must be different")
		return
	}

	// Проверка поддерживаемых валют
	if !models.IsSupportedCurrencyFromList(from, h.supportedCurrencies) {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error",
			fmt.Sprintf("Currency '%s' is not supported. Supported currencies: %v", from, h.supportedCurrencies))
		return
	}
	if !models.IsSupportedCurrencyFromList(to, h.supportedCurrencies) {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error",
			fmt.Sprintf("Currency '%s' is not supported. Supported currencies: %v", to, h.supportedCurrencies))
		return
	}

	// Создаем или получаем существующий pending запрос (идемпотентность)
	quoteRequest, err := h.db.CreateOrGetPendingQuoteRequest(from, to)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"from": from,
			"to":   to,
		}).Error("Failed to create or get quote request")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal error", "Failed to create quote request")
		return
	}

	response := models.UpdateQuoteResponse{
		ID:     quoteRequest.ID,
		From:   quoteRequest.From,
		To:     quoteRequest.To,
		Status: quoteRequest.Status,
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": quoteRequest.ID,
		"from":       from,
		"to":         to,
	}).Info("Quote update request created or retrieved")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// @Summary Получить котировку по ID запроса
// @Description Возвращает котировку валютной пары по ID запроса на обновление
// @Tags quotes
// @Accept json
// @Produce json
// @Param id path string true "ID запроса на обновление котировки"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /quotes/{id} [get]
func (h *Handler) GetQuoteByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	requestID := vars["id"]

	if strings.TrimSpace(requestID) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "Request ID is required")
		return
	}

	// Получаем запрос на обновление котировки
	quoteRequest, err := h.db.GetQuoteRequest(requestID)
	if err != nil {
		h.logger.WithError(err).WithField("request_id", requestID).Error("Failed to get quote request")
		h.writeErrorResponse(w, http.StatusNotFound, "Not found", "Quote request not found")
		return
	}

	// Проверяем статус запроса
	if quoteRequest.Status != "completed" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Request not completed",
			"Quote request is not completed yet. Status: "+quoteRequest.Status)
		return
	}

	// Получаем котировку
	quote, err := h.db.GetQuote(quoteRequest.From, quoteRequest.To)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"request_id": requestID,
			"from":       quoteRequest.From,
			"to":         quoteRequest.To,
		}).Error("Failed to get quote")
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal error", "Failed to get quote")
		return
	}

	response := models.QuoteResponse{
		ID:        quote.ID,
		From:      quote.From,
		To:        quote.To,
		Rate:      quote.Rate,
		UpdatedAt: quote.UpdatedAt,
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"from":       quote.From,
		"to":         quote.To,
		"rate":       quote.Rate,
	}).Info("Quote retrieved by ID")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// @Summary Получить последнюю котировку валютной пары
// @Description Возвращает последнее значение котировки для указанной валютной пары
// @Tags quotes
// @Accept json
// @Produce json
// @Param from query string true "Базовая валюта (например, EUR)"
// @Param to query string true "Котируемая валюта (например, MXN)"
// @Success 200 {object} models.QuoteResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /quotes/latest [get]
func (h *Handler) GetLatestQuote(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры из query string
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if strings.TrimSpace(from) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "From currency is required")
		return
	}
	if strings.TrimSpace(to) == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error", "To currency is required")
		return
	}

	// Нормализуем валюты
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))

	// Проверка поддерживаемых валют
	if !models.IsSupportedCurrencyFromList(from, h.supportedCurrencies) {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error",
			fmt.Sprintf("Currency '%s' is not supported. Supported currencies: %v", from, h.supportedCurrencies))
		return
	}
	if !models.IsSupportedCurrencyFromList(to, h.supportedCurrencies) {
		h.writeErrorResponse(w, http.StatusBadRequest, "Validation error",
			fmt.Sprintf("Currency '%s' is not supported. Supported currencies: %v", to, h.supportedCurrencies))
		return
	}

	// Получаем последнюю котировку
	quote, err := h.db.GetQuote(from, to)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"from": from,
			"to":   to,
		}).Error("Failed to get latest quote")
		h.writeErrorResponse(w, http.StatusNotFound, "Not found", "Quote not found for currency pair: "+from+"/"+to)
		return
	}

	response := models.QuoteResponse{
		ID:        quote.ID,
		From:      quote.From,
		To:        quote.To,
		Rate:      quote.Rate,
		UpdatedAt: quote.UpdatedAt,
	}

	h.logger.WithFields(logrus.Fields{
		"from": from,
		"to":   to,
		"rate": quote.Rate,
	}).Info("Latest quote retrieved")

	h.writeJSONResponse(w, http.StatusOK, response)
}

// @Summary Health check
// @Description Проверка состояния сервиса
// @Tags system
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"service":   "currency-quote-service",
		"timestamp": "2025-09-28T04:32:27Z",
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Записываем JSON ответ
func (h *Handler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.WithError(err).Error("Failed to encode JSON response")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Записываем JSON ответ с ошибкой
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, error, message string) {
	response := models.ErrorResponse{
		Error:   error,
		Message: message,
	}

	h.writeJSONResponse(w, statusCode, response)
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/quotes/update", h.UpdateQuote).Methods("POST")
	router.HandleFunc("/quotes/latest", h.GetLatestQuote).Methods("GET")
	router.HandleFunc("/quotes/{id}", h.GetQuoteByID).Methods("GET")
	router.HandleFunc("/health", h.Health).Methods("GET")
}
