package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"go_plata_task_v2/internal/models"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Интерфейс для работы с базой данных
type DatabaseInterface interface {
	CreateQuoteRequest(from, to string) (*models.QuoteRequest, error)
	GetQuoteRequest(id string) (*models.QuoteRequest, error)
	GetQuote(from, to string) (*models.Quote, error)
	UpdateQuoteRequestStatus(id, status string) error
	UpsertQuote(from, to string, rate float64) error
	GetPendingQuoteRequests() ([]*models.QuoteRequest, error)
	Close() error
}

// Мок для базы данных
type MockDB struct {
	mock.Mock
}

func (m *MockDB) CreateQuoteRequest(from, to string) (*models.QuoteRequest, error) {
	args := m.Called(from, to)
	return args.Get(0).(*models.QuoteRequest), args.Error(1)
}

func (m *MockDB) CreateOrGetPendingQuoteRequest(from, to string) (*models.QuoteRequest, error) {
	args := m.Called(from, to)
	return args.Get(0).(*models.QuoteRequest), args.Error(1)
}

func (m *MockDB) GetPendingQuoteRequestByPair(from, to string) (*models.QuoteRequest, error) {
	args := m.Called(from, to)
	return args.Get(0).(*models.QuoteRequest), args.Error(1)
}

func (m *MockDB) GetQuoteRequest(id string) (*models.QuoteRequest, error) {
	args := m.Called(id)
	return args.Get(0).(*models.QuoteRequest), args.Error(1)
}

func (m *MockDB) GetQuote(from, to string) (*models.Quote, error) {
	args := m.Called(from, to)
	return args.Get(0).(*models.Quote), args.Error(1)
}

func (m *MockDB) UpdateQuoteRequestStatus(id, status string) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDB) UpsertQuote(from, to string, rate float64) error {
	args := m.Called(from, to, rate)
	return args.Error(0)
}

func (m *MockDB) GetPendingQuoteRequests() ([]*models.QuoteRequest, error) {
	args := m.Called()
	return args.Get(0).([]*models.QuoteRequest), args.Error(1)
}

func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestUpdateQuote(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    models.UpdateQuoteRequest
		mockSetup      func(*MockDB)
		expectedStatus int
		expectedError  string
	}{
		{
			name: "Valid request",
			requestBody: models.UpdateQuoteRequest{
				From: "EUR",
				To:   "USD",
			},
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("CreateOrGetPendingQuoteRequest", "EUR", "USD").Return(&models.QuoteRequest{
					ID:     "123",
					From:   "EUR",
					To:     "USD",
					Status: "pending",
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Empty from currency",
			requestBody: models.UpdateQuoteRequest{
				From: "",
				To:   "USD",
			},
			mockSetup:      func(mockDB *MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "From currency is required",
		},
		{
			name: "Empty to currency",
			requestBody: models.UpdateQuoteRequest{
				From: "EUR",
				To:   "",
			},
			mockSetup:      func(mockDB *MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "To currency is required",
		},
		{
			name: "Same currencies",
			requestBody: models.UpdateQuoteRequest{
				From: "EUR",
				To:   "EUR",
			},
			mockSetup:      func(mockDB *MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "From and To currencies must be different",
		},
		{
			name: "Unsupported from currency",
			requestBody: models.UpdateQuoteRequest{
				From: "GBP",
				To:   "USD",
			},
			mockSetup:      func(mockDB *MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Currency 'GBP' is not supported",
		},
		{
			name: "Unsupported to currency",
			requestBody: models.UpdateQuoteRequest{
				From: "USD",
				To:   "GBP",
			},
			mockSetup:      func(mockDB *MockDB) {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Currency 'GBP' is not supported",
		},
		{
			name: "Database error",
			requestBody: models.UpdateQuoteRequest{
				From: "EUR",
				To:   "USD",
			},
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("CreateOrGetPendingQuoteRequest", "EUR", "USD").Return((*models.QuoteRequest)(nil), assert.AnError)
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			tt.mockSetup(mockDB)

			logger := logrus.New()
			handler := &Handler{
				db:                  mockDB,
				logger:              logger,
				supportedCurrencies: []string{"USD", "EUR", "MXN"},
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/quotes/update", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.UpdateQuote(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedError != "" {
				var errorResp models.ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &errorResp)
				assert.NoError(t, err)
				assert.Contains(t, errorResp.Message, tt.expectedError)
			}

			mockDB.AssertExpectations(t)
		})
	}
}

func TestGetQuoteByID(t *testing.T) {
	tests := []struct {
		name           string
		requestID      string
		mockSetup      func(*MockDB)
		expectedStatus int
	}{
		{
			name:      "Valid request",
			requestID: "123",
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("GetQuoteRequest", "123").Return(&models.QuoteRequest{
					ID:     "123",
					From:   "EUR",
					To:     "USD",
					Status: "completed",
				}, nil)
				mockDB.On("GetQuote", "EUR", "USD").Return(&models.Quote{
					ID:   "456",
					From: "EUR",
					To:   "USD",
					Rate: 1.1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "Request not found",
			requestID: "999",
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("GetQuoteRequest", "999").Return((*models.QuoteRequest)(nil), assert.AnError)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:      "Request not completed",
			requestID: "123",
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("GetQuoteRequest", "123").Return(&models.QuoteRequest{
					ID:     "123",
					From:   "EUR",
					To:     "USD",
					Status: "pending",
				}, nil)
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			tt.mockSetup(mockDB)

			logger := logrus.New()
			handler := &Handler{
				db:                  mockDB,
				logger:              logger,
				supportedCurrencies: []string{"USD", "EUR", "MXN"},
			}

			req := httptest.NewRequest("GET", "/quotes/"+tt.requestID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.requestID})

			rr := httptest.NewRecorder()
			handler.GetQuoteByID(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockDB.AssertExpectations(t)
		})
	}
}

func TestGetLatestQuote(t *testing.T) {
	tests := []struct {
		name           string
		from           string
		to             string
		mockSetup      func(*MockDB)
		expectedStatus int
	}{
		{
			name: "Valid request",
			from: "EUR",
			to:   "USD",
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("GetQuote", "EUR", "USD").Return(&models.Quote{
					ID:   "456",
					From: "EUR",
					To:   "USD",
					Rate: 1.1,
				}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Quote not found",
			from: "EUR",
			to:   "MXN",
			mockSetup: func(mockDB *MockDB) {
				mockDB.On("GetQuote", "EUR", "MXN").Return((*models.Quote)(nil), assert.AnError)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Empty from currency",
			from: "",
			to:   "USD",
			mockSetup: func(mockDB *MockDB) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty to currency",
			from: "EUR",
			to:   "",
			mockSetup: func(mockDB *MockDB) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unsupported from currency",
			from: "GBP",
			to:   "USD",
			mockSetup: func(mockDB *MockDB) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Unsupported to currency",
			from: "USD",
			to:   "GBP",
			mockSetup: func(mockDB *MockDB) {
				// No mock setup needed for validation error
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDB)
			tt.mockSetup(mockDB)

			logger := logrus.New()
			handler := &Handler{
				db:                  mockDB,
				logger:              logger,
				supportedCurrencies: []string{"USD", "EUR", "MXN"},
			}

			req := httptest.NewRequest("GET", "/quotes/latest?from="+tt.from+"&to="+tt.to, nil)

			rr := httptest.NewRecorder()
			handler.GetLatestQuote(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			mockDB.AssertExpectations(t)
		})
	}
}
