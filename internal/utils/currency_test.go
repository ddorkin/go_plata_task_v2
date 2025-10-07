package utils

import (
	"testing"
)

func TestCalculateExchangeRate(t *testing.T) {
	// Тестовые данные - курсы относительно USD (только поддерживаемые валюты)
	usdRates := map[string]float64{
		"USD": 1.0,
		"EUR": 0.85,
		"MXN": 18.5,
	}

	tests := []struct {
		name     string
		from     string
		to       string
		expected float64
		wantErr  bool
	}{
		{
			name:     "USD to EUR",
			from:     "USD",
			to:       "EUR",
			expected: 0.85,
			wantErr:  false,
		},
		{
			name:     "EUR to USD",
			from:     "EUR",
			to:       "USD",
			expected: 1.0 / 0.85, // ≈ 1.176
			wantErr:  false,
		},
		{
			name:     "USD to MXN",
			from:     "USD",
			to:       "MXN",
			expected: 18.5,
			wantErr:  false,
		},
		{
			name:     "MXN to USD",
			from:     "MXN",
			to:       "USD",
			expected: 1.0 / 18.5, // ≈ 0.054
			wantErr:  false,
		},
		{
			name:     "EUR to MXN (cross rate)",
			from:     "EUR",
			to:       "MXN",
			expected: 18.5 / 0.85, // ≈ 21.765
			wantErr:  false,
		},
		{
			name:     "MXN to EUR (cross rate)",
			from:     "MXN",
			to:       "EUR",
			expected: 0.85 / 18.5, // ≈ 0.046
			wantErr:  false,
		},
		{
			name:     "USD to USD (same currency)",
			from:     "USD",
			to:       "USD",
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "EUR to EUR (same currency)",
			from:     "EUR",
			to:       "EUR",
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "MXN to MXN (same currency)",
			from:     "MXN",
			to:       "MXN",
			expected: 1.0,
			wantErr:  false,
		},
		{
			name:     "Non-existent from currency",
			from:     "CAD",
			to:       "USD",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Non-existent to currency",
			from:     "USD",
			to:       "CAD",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Both currencies non-existent",
			from:     "CAD",
			to:       "AUD",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateExchangeRate(tt.from, tt.to, usdRates)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CalculateExchangeRate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CalculateExchangeRate() unexpected error: %v", err)
				return
			}

			// Используем приблизительное сравнение для float64
			if !isApproximatelyEqual(result, tt.expected, 0.0001) {
				t.Errorf("CalculateExchangeRate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateExchangeRate_EdgeCases(t *testing.T) {
	t.Run("Empty rates map", func(t *testing.T) {
		_, err := CalculateExchangeRate("USD", "EUR", map[string]float64{})
		if err == nil {
			t.Error("Expected error for empty rates map")
		}
	})

	t.Run("Nil rates map", func(t *testing.T) {
		_, err := CalculateExchangeRate("USD", "EUR", nil)
		if err == nil {
			t.Error("Expected error for nil rates map")
		}
	})

	t.Run("Very small rates", func(t *testing.T) {
		rates := map[string]float64{
			"USD": 1.0,
			"VND": 0.000043, // Вьетнамский донг
		}

		result, err := CalculateExchangeRate("USD", "VND", rates)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := 0.000043
		if !isApproximatelyEqual(result, expected, 0.0000001) {
			t.Errorf("CalculateExchangeRate() = %v, expected %v", result, expected)
		}
	})

	t.Run("Very large rates", func(t *testing.T) {
		rates := map[string]float64{
			"USD": 1.0,
			"IRR": 42000.0, // Иранский риал
		}

		result, err := CalculateExchangeRate("USD", "IRR", rates)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := 42000.0
		if !isApproximatelyEqual(result, expected, 0.1) {
			t.Errorf("CalculateExchangeRate() = %v, expected %v", result, expected)
		}
	})
}

func TestCalculateExchangeRate_Precision(t *testing.T) {
	// Тест на точность вычислений с реальными курсами (только поддерживаемые валюты)
	rates := map[string]float64{
		"USD": 1.0,
		"EUR": 0.8534,
		"MXN": 18.4567,
	}

	t.Run("High precision cross rate EUR to MXN", func(t *testing.T) {
		result, err := CalculateExchangeRate("EUR", "MXN", rates)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := 18.4567 / 0.8534 // ≈ 21.627
		if !isApproximatelyEqual(result, expected, 0.001) {
			t.Errorf("CalculateExchangeRate() = %v, expected %v", result, expected)
		}
	})

	t.Run("High precision cross rate MXN to EUR", func(t *testing.T) {
		result, err := CalculateExchangeRate("MXN", "EUR", rates)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := 0.8534 / 18.4567 // ≈ 0.046
		if !isApproximatelyEqual(result, expected, 0.001) {
			t.Errorf("CalculateExchangeRate() = %v, expected %v", result, expected)
		}
	})
}

// Проверяем, что два float64 значения приблизительно равны
func isApproximatelyEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
