package utils

import "fmt"

// Вычисляем курс валютной пары используя курсы относительно USD
func CalculateExchangeRate(from, to string, usdRates map[string]float64) (float64, error) {
	fromRate, fromExists := usdRates[from]
	if !fromExists {
		return 0, fmt.Errorf("currency %s not found in rates", from)
	}

	toRate, toExists := usdRates[to]
	if !toExists {
		return 0, fmt.Errorf("currency %s not found in rates", to)
	}

	// Вычисляем курс пары from/to
	// API возвращает курсы относительно USD
	var rate float64
	if from == "USD" {
		rate = toRate
	} else if to == "USD" {
		rate = 1.0 / fromRate
	} else {
		rate = toRate / fromRate
	}

	return rate, nil
}
