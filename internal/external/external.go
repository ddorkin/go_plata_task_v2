package external

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go_plata_task_v2/internal/config"
	"go_plata_task_v2/internal/models"

	"github.com/sirupsen/logrus"
)

// Клиент для работы с внешним API
type Client struct {
	httpClient          *http.Client
	baseURL             string
	apiKey              string
	supportedCurrencies []string
	logger              *logrus.Logger
}

// Создаём новый клиент для внешнего API
func New(cfg *config.ExternalConfig, supportedCurrencies []string, logger *logrus.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		baseURL:             cfg.BaseURL,
		apiKey:              cfg.APIKey,
		supportedCurrencies: supportedCurrencies,
		logger:              logger,
	}
}

// Получаем курсы всех валют относительно USD одним запросом
func (c *Client) GetMultipleExchangeRates(currencies []string) (map[string]float64, error) {
	if len(currencies) == 0 {
		return make(map[string]float64), nil
	}

	// Убираем дубликаты, исключаем USD так как он уже указан как base
	uniqueCurrencies := make(map[string]bool)
	for _, currency := range currencies {
		if currency != "USD" { // Исключаем USD из symbols, так как base=USD
			uniqueCurrencies[currency] = true
		}
	}

	// Формируем список символов для запроса
	var symbols []string
	for currency := range uniqueCurrencies {
		symbols = append(symbols, currency)
	}

	// Формируем URL для batch запроса
	url := fmt.Sprintf("%s/latest?base=USD&symbols=%s", c.baseURL, strings.Join(symbols, ","))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Добавляем API ключ если он есть
	if c.apiKey != "" {
		req.Header.Set("apikey", c.apiKey)
	}

	req.Header.Set("User-Agent", "Currency-Quote-Service/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.WithField("response_body", string(body)).Debug("External API batch response")

	var apiResp models.ExternalAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API returned success=false")
	}

	// Добавляем USD в результат (курс USD к самому себе = 1.0)
	apiResp.Rates["USD"] = 1.0

	c.logger.WithFields(logrus.Fields{
		"currencies":  symbols,
		"rates_count": len(apiResp.Rates),
	}).Info("Successfully retrieved batch exchange rates")

	return apiResp.Rates, nil
}
