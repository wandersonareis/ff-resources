package testcommon

import (
	services "ffresources/internal/families/ffx/service"
)

// MockNotifier implementa services.INotificationService para os testes:
// guarda as notificações em memória, sem publicar eventos nem tocar no
// frontend.
type MockNotifier struct {
	Notifications []services.Notification
}

// NewMockNotifier cria um notificador de teste vazio.
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{Notifications: []services.Notification{}}
}

// Notify registra a notificação com a severidade pedida.
func (m *MockNotifier) Notify(severity services.Severity, message string) {
	m.Notifications = append(m.Notifications, services.Notification{
		Message:  message,
		Severity: severity.String(),
	})
}

// NotifyError registra um erro.
func (m *MockNotifier) NotifyError(err error) {
	m.Notify(services.SeverityError, err.Error())
}

// NotifyInfo registra um aviso informativo.
func (m *MockNotifier) NotifyInfo(message string) {
	m.Notify(services.SeverityInfo, message)
}

// NotifyWarn registra um aviso.
func (m *MockNotifier) NotifyWarn(message string) {
	m.Notify(services.SeverityWarn, message)
}

// NotifySuccess registra um sucesso.
func (m *MockNotifier) NotifySuccess(message string) {
	m.Notify(services.SeveritySuccess, message)
}
