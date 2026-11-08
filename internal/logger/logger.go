package logger

import "go.uber.org/zap"

// New создаёт production-журнал zap с выводом структурированных сообщений.
// После завершения работы вызывающий код должен вызвать Sync.
func New() (*zap.Logger, error) {
	return zap.NewProduction()
}
