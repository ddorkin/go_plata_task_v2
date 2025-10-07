package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger представляет структурированный логгер
type Logger struct {
	*logrus.Logger
}

// New создает новый экземпляр логгера
func New(level string) *Logger {
	logger := logrus.New()

	// Настройка формата вывода
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Настройка уровня логирования
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	logger.SetLevel(logLevel)

	// Вывод в stdout
	logger.SetOutput(os.Stdout)

	return &Logger{logger}
}

// WithField добавляет поле к логгеру
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields добавляет несколько полей к логгеру
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithError добавляет ошибку к логгеру
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// Info логирует сообщение уровня Info
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

// Error логирует сообщение уровня Error
func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

// Fatal логирует сообщение уровня Fatal и завершает программу
func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

// Debug логирует сообщение уровня Debug
func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

// Warn логирует сообщение уровня Warn
func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}
