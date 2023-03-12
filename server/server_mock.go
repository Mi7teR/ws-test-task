package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"ws-test-task/entity"
)

type authState struct {
	authorized bool
	at         time.Time
}

type Writer struct {
	out chan interface{}

	subMu      sync.RWMutex
	subscribed map[string]struct{}

	ctx         context.Context
	ctxCancelFn func()

	authMu sync.RWMutex
	auth   authState
}

var (
	ErrDisconnected = fmt.Errorf("disconnected")
)

func NewWriter() *Writer {
	ctx, cancel := context.WithCancel(context.Background())

	return &Writer{
		out:         make(chan interface{}, 10),
		subscribed:  make(map[string]struct{}),
		ctx:         ctx,
		ctxCancelFn: cancel,
	}
}

func (w *Writer) streamSymbol(symbol string) {
	tk := time.NewTicker(time.Millisecond * 500)
	defer tk.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-tk.C:
			w.out <- entity.MethodResponse{
				Method: entity.MethodExecutions,
				Data: map[string]string{
					"symbol": symbol,
					"price":  fmt.Sprintf("%d", time.Now().UnixMilli()),
					"amount": "105",
				},
			}
		}
	}

}

// Send отправляет сообщение по WebSocket. Для упрощения реализация принимает сразу MethodRequest вместо []byte
func (w *Writer) Send(m entity.MethodRequest) error {
	select {
	case <-w.ctx.Done():
		return ErrDisconnected
	default:

	}

	// TODO: писать в w.out не безопасно
	switch m.Method {
	case entity.MethodExecutions:
		symbol := m.Args["symbol"]

		w.subMu.RLock()
		_, ok := w.subscribed[symbol]
		w.subMu.RUnlock()
		if !ok {

			w.authMu.RLock()
			isAuth := w.auth.authorized
			w.authMu.RUnlock()

			if !isAuth {
				w.out <- entity.StatusResponse{
					ReqID:  m.ReqID,
					Status: false,
					Error:  "this method isn't accesible without authorization",
				}

				return nil
			}

			w.subMu.Lock()
			w.subscribed[symbol] = struct{}{}
			w.subMu.Unlock()

			w.out <- entity.StatusResponse{
				ReqID:  m.ReqID,
				Status: true,
				Error:  "",
			}

			go w.streamSymbol(symbol)

			return nil
		}

		w.out <- entity.StatusResponse{
			ReqID:  m.ReqID,
			Status: false,
			Error:  "already subscribed",
		}

	case entity.MethodAuth:
		if err := w.authCheck(m); err == nil {

			w.authMu.Lock()
			w.auth = authState{
				authorized: true,
				at:         time.Now(),
			}
			w.authMu.Unlock()

			w.out <- entity.StatusResponse{
				ReqID:  m.ReqID,
				Status: true,
				Error:  "",
			}

		} else {
			w.out <- entity.StatusResponse{
				ReqID:  m.ReqID,
				Status: false,
				Error:  err.Error(),
			}
		}

	default:
		// just ignore
		return nil
	}

	return nil
}

func (w *Writer) Read() <-chan []byte {
	byteCh := make(chan []byte)

	go func() {
		defer close(byteCh)

		for {
			select {
			case item := <-w.out:

				b, err := json.Marshal(item)
				if err != nil {
					fmt.Printf("could not marshal object: %v\n", err)
				}

				byteCh <- b
			case <-w.ctx.Done():
				return
			}
		}
	}()

	return byteCh
}

func (w *Writer) authCheck(a entity.MethodRequest) error {
	if a.Method != entity.MethodAuth {
		return fmt.Errorf("incorrect method: %s", a.Method)
	}

	if a.Args["login"] != "foo" || a.Args["password"] != "bar" {
		return fmt.Errorf("wrong creds")
	}

	return nil
}

func (w *Writer) Stop() {
	w.ctxCancelFn()

	// close(w.out)
}

func (w *Writer) Run() error {
	tkExpiring := time.NewTicker(time.Second * 10)
	defer tkExpiring.Stop()

	for {
		select {
		case <-tkExpiring.C:
			w.authMu.RLock()
			auth := w.auth
			w.authMu.RUnlock()

			if !auth.authorized {
				continue
			}

			timeDiff := time.Now().Sub(auth.at)
			if timeDiff > time.Minute {
				w.Stop()
				return ErrDisconnected
			}

			if timeDiff > time.Second*20 {
				w.out <- entity.MethodResponse{
					Method: entity.MethodAuthExpiring,
				}
			}
		}
	}
}
