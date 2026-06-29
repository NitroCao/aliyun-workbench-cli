package workbench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type terminalSize struct {
	Rows int
	Cols int
}

type interactiveSession struct {
	ticket     string
	region     string
	instanceID string
	login      *LoginResult
	ws         *wsConn
	errCh      chan error
}

func RunInteractiveSession(ctx context.Context, ticket, region, instanceID string, login *LoginResult) error {
	session := &interactiveSession{
		ticket:     ticket,
		region:     region,
		instanceID: instanceID,
		login:      login,
	}
	return session.run(ctx)
}

func (s *interactiveSession) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ws, err := dialWorkbenchWebSocket(ctx, s.ticket)
	if err != nil {
		return err
	}
	s.ws = ws
	defer s.ws.closeAndLog()
	go s.closeWebSocketOnCancel(ctx)

	terminalState, err := makeRaw(ctx)
	if err == nil {
		defer restoreTerminal(ctx, terminalState)
	}

	size := currentTerminalSize(ctx)
	if err := s.sendConnect(ctx, size); err != nil {
		return err
	}

	s.errCh = make(chan error, 3)
	go s.readLoop(ctx)
	go s.stdinLoop(ctx)
	go s.resizeLoop(ctx)

	return s.wait()
}

func (s *interactiveSession) wait() error {
	if err := <-s.errCh; err != nil && !errors.Is(err, errWebSocketClosed) && !errors.Is(err, io.EOF) && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (s *interactiveSession) closeWebSocketOnCancel(ctx context.Context) {
	<-ctx.Done()
	s.ws.closeAndLog()
}

func (s *interactiveSession) sendConnect(ctx context.Context, size terminalSize) error {
	payload := terminalSizePayload(size)
	maps.Copy(payload, s.login.Root)

	payload["protocol"] = "ssh"
	payload["cwd"] = ""
	payload["resourceURI"] = fmt.Sprintf("from=ecs&instanceType=ecs&regionId=%s&instanceId=%s&resourceGroupId=&language=zh-CN", s.region, s.instanceID)

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.ws.writeText(ctx, encodeWB("connect", body))
}

func (s *interactiveSession) sendSize(ctx context.Context, size terminalSize) error {
	payload := terminalSizePayload(size)
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.ws.writeText(ctx, encodeWB("size", body))
}

func terminalSizePayload(size terminalSize) map[string]any {
	cols, rows := size.Cols, size.Rows
	return map[string]any{
		"terminalColumns": cols,
		"terminalLines":   rows,
		"width":           cols * 8,
		"height":          rows * 14,
		"offsetLeft":      0,
		"offsetTop":       0,
		"pageWidth":       cols * 8,
		"pageHeight":      rows * 14,
		"windowWidth":     cols * 8,
		"windowHeight":    rows * 14,
		"windowLeft":      0,
		"windowTop":       0,
		"screenWidth":     cols * 8,
		"screenHeight":    rows * 14,
		"dpi":             96,
		"lineHeight":      1,
	}
}

func (s *interactiveSession) readLoop(ctx context.Context) {
	for {
		data, err := s.ws.readText(ctx)
		if err != nil {
			s.errCh <- err
			return
		}
		messages, err := parseWB(data)
		if err != nil {
			s.errCh <- err
			return
		}
		for _, msg := range messages {
			if err := s.handleMessage(ctx, msg); err != nil {
				s.errCh <- err
				return
			}
		}
	}
}

func (s *interactiveSession) handleMessage(ctx context.Context, msg wbMessage) error {
	switch msg.Command {
	case "text":
		for _, part := range msg.Parts {
			_, _ = os.Stdout.Write(part)
		}
	case "heartbeat":
		return s.ws.writeText(ctx, encodeWB("delayping", []byte(strconv.FormatInt(time.Now().UnixMilli(), 10))))
	case "notification", "frontdelay", "backdelay":
		return nil
	default:
		return nil
	}
	return nil
}

func (s *interactiveSession) stdinLoop(ctx context.Context) {
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			if werr := s.ws.writeText(ctx, encodeWB("text", buf[:n])); werr != nil {
				s.errCh <- werr
				return
			}
		}
		if err != nil {
			s.errCh <- err
			return
		}
	}
}

func (s *interactiveSession) resizeLoop(ctx context.Context) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	defer signal.Stop(ch)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			_ = s.sendSize(ctx, currentTerminalSize(ctx))
		}
	}
}

func currentTerminalSize(ctx context.Context) terminalSize {
	cmd := exec.CommandContext(ctx, "stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return terminalSize{Rows: 24, Cols: 80}
	}
	parts := strings.Fields(string(out))
	if len(parts) != 2 {
		return terminalSize{Rows: 24, Cols: 80}
	}
	rows, err1 := strconv.Atoi(parts[0])
	cols, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || rows <= 0 || cols <= 0 {
		return terminalSize{Rows: 24, Cols: 80}
	}
	return terminalSize{Rows: rows, Cols: cols}
}

func makeRaw(ctx context.Context) (string, error) {
	stateCmd := exec.CommandContext(ctx, "stty", "-g")
	stateCmd.Stdin = os.Stdin
	stateBytes, err := stateCmd.Output()
	if err != nil {
		return "", err
	}
	state := strings.TrimSpace(string(stateBytes))

	rawCmd := exec.CommandContext(ctx, "stty", "raw", "-echo")
	rawCmd.Stdin = os.Stdin
	if err := rawCmd.Run(); err != nil {
		return "", err
	}

	return state, nil
}

func restoreTerminal(ctx context.Context, state string) {
	restoreCmd := exec.CommandContext(ctx, "stty", state) //nolint:gosec // state is captured from `stty -g` for terminal restoration.
	restoreCmd.Stdin = os.Stdin
	_ = restoreCmd.Run()
}
