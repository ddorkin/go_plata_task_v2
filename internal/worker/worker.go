package worker

import (
	"context"
	"time"

	"go_plata_task_v2/internal/database"
	"go_plata_task_v2/internal/external"
	"go_plata_task_v2/internal/models"
	"go_plata_task_v2/internal/utils"

	"github.com/sirupsen/logrus"
)

// Worker представляет фоновый воркер для обновления котировок
type Worker struct {
	db          *database.DB
	externalAPI *external.Client
	logger      *logrus.Logger
	ticker      *time.Ticker
	done        chan bool
	interval    time.Duration
}

// Создаём новый воркер
func New(db *database.DB, externalAPI *external.Client, logger *logrus.Logger, interval time.Duration) *Worker {
	return &Worker{
		db:          db,
		externalAPI: externalAPI,
		logger:      logger,
		done:        make(chan bool),
		interval:    interval,
	}
}

// Запускаем воркер
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting quote update worker")

	// Запускаем воркер с настраиваемым интервалом
	w.ticker = time.NewTicker(w.interval)

	// Выполняем первую проверку сразу
	go w.processPendingRequests()

	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.processPendingRequests()
			case <-w.done:
				w.logger.Info("Worker stopped")
				return
			case <-ctx.Done():
				w.logger.Info("Worker context cancelled")
				return
			}
		}
	}()
}

// Стопаем воркер
func (w *Worker) Stop() {
	if w.ticker != nil {
		w.ticker.Stop()
	}
	w.done <- true
}

// Обрабатываем ожидающие запросы на обновление котировок
func (w *Worker) processPendingRequests() {
	w.logger.Debug("Processing pending quote requests")

	// Получаем все ожидающие запросы
	requests, err := w.db.GetPendingQuoteRequests()
	if err != nil {
		w.logger.WithError(err).Error("Failed to get pending quote requests")
		return
	}

	if len(requests) == 0 {
		w.logger.Debug("No pending quote requests found")
		return
	}

	w.logger.WithField("count", len(requests)).Info("Found pending quote requests")

	// Собираем все уникальные валюты из запросов
	currencies := w.extractUniqueCurrencies(requests)

	// Получаем все курсы одним batch запросом
	usdRates, err := w.externalAPI.GetMultipleExchangeRates(currencies)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get batch exchange rates")
		// Помечаем все запросы как failed
		w.markAllRequestsAsFailed(requests, err)
		return
	}

	// Группируем запросы по валютным парам
	currencyPairMap := make(map[string][]*models.QuoteRequest)
	for _, req := range requests {
		pair := req.From + "/" + req.To
		currencyPairMap[pair] = append(currencyPairMap[pair], req)
	}

	// Обрабатываем каждую валютную пару с использованием полученных курсов
	for pair, reqs := range currencyPairMap {
		w.processCurrencyPairWithRates(pair, reqs, usdRates)
	}
}

// Извлекаем все уникальные валюты из запросов
func (w *Worker) extractUniqueCurrencies(requests []*models.QuoteRequest) []string {
	currencies := make(map[string]bool)
	for _, req := range requests {
		currencies[req.From] = true
		currencies[req.To] = true
	}

	var result []string
	for currency := range currencies {
		result = append(result, currency)
	}

	return result
}

// Помечаем все запросы как failed
func (w *Worker) markAllRequestsAsFailed(requests []*models.QuoteRequest, err error) {
	for _, req := range requests {
		if updateErr := w.db.UpdateQuoteRequestStatus(req.ID, "failed"); updateErr != nil {
			w.logger.WithError(updateErr).WithField("request_id", req.ID).Error("Failed to update request status to failed")
		}
	}
}

// Обрабатываем валютную пару используя предварительно полученные курсы
func (w *Worker) processCurrencyPairWithRates(pair string, requests []*models.QuoteRequest, usdRates map[string]float64) {
	w.logger.WithField("pair", pair).Debug("Processing currency pair requests with pre-fetched rates")

	// Обновляем статус всех запросов на "processing"
	for _, req := range requests {
		if err := w.db.UpdateQuoteRequestStatus(req.ID, "processing"); err != nil {
			w.logger.WithError(err).WithField("request_id", req.ID).Error("Failed to update request status to processing")
		}
	}

	from := requests[0].From
	to := requests[0].To

	// Вычисляем курс пары используя предварительно полученные курсы
	rate, err := utils.CalculateExchangeRate(from, to, usdRates)
	if err != nil {
		w.logger.WithError(err).WithFields(logrus.Fields{
			"pair": pair,
			"from": from,
			"to":   to,
		}).Error("Failed to calculate exchange rate")

		// Обновляем статус всех запросов на "failed"
		for _, req := range requests {
			if err := w.db.UpdateQuoteRequestStatus(req.ID, "failed"); err != nil {
				w.logger.WithError(err).WithField("request_id", req.ID).Error("Failed to update request status to failed")
			}
		}
		return
	}

	// Сохраняем котировку в базу данных
	if err := w.db.UpsertQuote(from, to, rate); err != nil {
		w.logger.WithError(err).WithFields(logrus.Fields{
			"pair": pair,
			"from": from,
			"to":   to,
		}).Error("Failed to save quote to database")

		// Обновляем статус всех запросов на "failed"
		for _, req := range requests {
			if err := w.db.UpdateQuoteRequestStatus(req.ID, "failed"); err != nil {
				w.logger.WithError(err).WithField("request_id", req.ID).Error("Failed to update request status to failed")
			}
		}
		return
	}

	// Обновляем статус всех запросов на "completed"
	for _, req := range requests {
		if err := w.db.UpdateQuoteRequestStatus(req.ID, "completed"); err != nil {
			w.logger.WithError(err).WithField("request_id", req.ID).Error("Failed to update request status to completed")
		}
	}

	w.logger.WithFields(logrus.Fields{
		"pair":  pair,
		"from":  from,
		"to":    to,
		"rate":  rate,
		"count": len(requests),
	}).Info("Successfully processed currency pair requests with batch rates")
}
