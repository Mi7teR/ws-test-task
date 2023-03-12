package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
	"ws-test-task/entity"

	"github.com/google/uuid"
)

var (
	ErrChannelNotFound = errors.New("channel not found")
	ErrConnClosed      = errors.New("connection closed")
)

type Client struct {
	c               map[string]chan entity.StatusResponse
	m               sync.RWMutex
	timeout         time.Duration
	login, password string

	sendFunc        func(request entity.MethodRequest) error
	responseHandler func(response entity.MethodResponse)
	statusHandler   func(response entity.StatusResponse)
	errorHandler    func(err error)
}

func NewClient(timeout time.Duration, login, password string) *Client {
	return &Client{timeout: timeout, login: login, password: password, c: make(map[string]chan entity.StatusResponse)}
}

func (client *Client) SetSendFunc(sendFunc func(request entity.MethodRequest) error) {
	client.sendFunc = sendFunc
}

func (client *Client) SetResponseHandler(responseHandler func(response entity.MethodResponse)) {
	client.responseHandler = responseHandler
}

func (client *Client) SetStatusHandler(statusHandler func(response entity.StatusResponse)) {
	client.statusHandler = statusHandler
}

func (client *Client) SetErrorHandler(errorHandler func(err error)) {
	client.errorHandler = errorHandler
}

func (client *Client) addChannel(id string, c chan entity.StatusResponse) {
	client.m.Lock()
	defer client.m.Unlock()

	client.c[id] = c
}

func (client *Client) removeChannel(id string) {
	client.m.Lock()
	defer client.m.Unlock()

	if c, ok := client.c[id]; ok {
		close(c)
		delete(client.c, id)
	}
}

func (client *Client) getChannel(id string) (chan entity.StatusResponse, error) {
	client.m.RLock()
	defer client.m.RUnlock()
	if c, ok := client.c[id]; ok {
		return c, nil
	}
	return nil, ErrChannelNotFound
}

func (client *Client) Listen(in <-chan []byte) error {
	for rawMessage := range in {
		if bytes.Contains(rawMessage, []byte("\"method\":")) {
			var res entity.MethodResponse

			err := json.Unmarshal(rawMessage, &res)
			if err != nil {
				client.errorHandler(fmt.Errorf("method response unmarshal error: %w", err))
				continue
			}

			if res.Method == entity.MethodAuthExpiring {
				err = client.Auth()
				if err != nil {
					client.errorHandler(fmt.Errorf("auth error: %w", err))
				}
				continue
			}

			client.responseHandler(res)
			continue
		}

		var res entity.StatusResponse

		err := json.Unmarshal(rawMessage, &res)
		if err != nil {
			client.errorHandler(fmt.Errorf("status response unmarshal error: %w", err))
			continue
		}

		client.statusHandler(res)
	}

	return ErrConnClosed
}

func (client *Client) Auth() error {
	requestID, err := client.send(entity.MethodAuth, map[string]string{
		"login":    client.login,
		"password": client.password,
	})
	if err != nil {
		return err
	}

	go client.handleStatusResponse(requestID)

	return nil
}

func (client *Client) handleStatusResponse(id string) {
	c, err := client.getChannel(id)
	if err != nil {
		client.errorHandler(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), client.timeout)
	defer cancel()

	select {
	case <-ctx.Done():
		client.errorHandler(fmt.Errorf("request %s context end: %w", id, ctx.Err()))
		return
	case res := <-c:
		client.statusHandler(res)
		return
	}
}

func (client *Client) send(method string, args map[string]string) (string, error) {
	requestID := uuid.New().String()
	c := make(chan entity.StatusResponse)

	client.addChannel(requestID, c)

	err := client.sendFunc(entity.MethodRequest{
		ReqID:  requestID,
		Method: method,
		Args:   args,
	})
	if err != nil {
		client.removeChannel(requestID)
		return "", err
	}

	return requestID, nil
}

func (client *Client) Subscribe(symbol string) (string, error) {
	args := map[string]string{
		"symbol": symbol,
	}

	requestID, err := client.send(entity.MethodExecutions, args)
	if err != nil {
		return "", err
	}

	return requestID, nil
}
