package entity

// MethodResponse стриминг сообщений после успешной подписки
type MethodResponse struct {
	Method string            `json:"method"`
	Data   map[string]string `json:"data"`
}
