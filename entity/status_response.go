package entity

// StatusResponse ответ от сервера на MethodRequest
type StatusResponse struct {
	// ReqID для матчинга между запрос <-> ответ
	ReqID string `json:"req_id"`
	// Status сигнализирует об успешности подписки
	Status bool `json:"status"`
	// Error детали ошибки, если Status = false
	Error string `json:"error"`
}
