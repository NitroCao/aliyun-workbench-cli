package workbench

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const workbenchWebSocketURL = "wss://ecs-workbench.aliyun.com/websocket/session/ssh"

var errWebSocketClosed = errors.New("websocket closed")

type wsConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func dialWorkbenchWebSocket(ctx context.Context, ticket string) (*wsConn, error) {
	headers := http.Header{}
	headers.Set("Accept-Language", "en-US,en;q=0.9")
	headers.Set("Cache-Control", "no-cache")
	headers.Set("Cookie", "login_aliyunid_ticket="+ticket)
	headers.Set("Origin", workbenchBase)
	headers.Set("Pragma", "no-cache")
	headers.Set("User-Agent", userAgent)

	dialer := websocket.Dialer{
		HandshakeTimeout: 20 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: "ecs-workbench.aliyun.com",
		},
	}
	logrus.WithFields(logrus.Fields{
		"url": workbenchWebSocketURL,
	}).Debug("websocket handshake request")

	conn, resp, err := dialer.DialContext(ctx, workbenchWebSocketURL, headers)
	if resp != nil {
		defer closeAndLog(resp.Body)
		logrus.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"url":    workbenchWebSocketURL,
		}).Debug("websocket handshake response")
	}

	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("websocket upgrade failed: %s: %w", resp.Status, err)
		}
		return nil, err
	}

	return &wsConn{conn: conn}, nil
}

func logWebSocketFrame(direction, kind string, payload []byte) {
	if !logrus.IsLevelEnabled(logrus.DebugLevel) {
		return
	}
	fields := logrus.Fields{
		"direction": direction,
		"kind":      kind,
		"length":    len(payload),
	}
	if kind == "text" {
		messages, err := parseWB(payload)
		if err != nil {
			fields["parse_error"] = err.Error()
		} else {
			commands := make([]string, 0, len(messages))
			for _, msg := range messages {
				commands = append(commands, msg.Command)
			}
			fields["commands"] = commands
		}
	}
	logrus.WithFields(fields).Debug("websocket frame")
}

func (c *wsConn) close() error {
	logWebSocketFrame("out", "close", nil)
	deadline := time.Now().Add(time.Second)
	_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), deadline)
	return c.conn.Close()
}

func (c *wsConn) closeAndLog() {
	if err := c.close(); err != nil {
		logrus.WithError(err).Debug("close websocket")
	}
}

func (c *wsConn) writeText(ctx context.Context, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	logWebSocketFrame("out", "text", payload)
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *wsConn) readText(ctx context.Context) ([]byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		messageType, payload, err := c.conn.ReadMessage()
		if err != nil {
			if isWebSocketClosedError(err) {
				return nil, errWebSocketClosed
			}
			return nil, err
		}

		if messageType == websocket.TextMessage {
			logWebSocketFrame("in", "text", payload)
			return payload, nil
		}
		if messageType == websocket.BinaryMessage {
			logWebSocketFrame("in", "binary", payload)
			return nil, fmt.Errorf("unsupported websocket message type %d", messageType)
		}
	}
}

func isWebSocketClosedError(err error) bool {
	if errors.Is(err, websocket.ErrCloseSent) {
		return true
	}
	var closeErr *websocket.CloseError
	return errors.As(err, &closeErr)
}
