package entity

// MethodRequest клиентский запрос на сервер
type MethodRequest struct {
	// ReqID уникальный в рамках сессий ID для последующего матчинга ответа к запросу
	ReqID string `json:"req_id"`
	// Method название метода, который вызывается
	Method string `json:"method"`
	// Args аргументы для вызова метода
	Args map[string]string `json:"args"`
}
