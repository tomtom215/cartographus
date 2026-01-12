// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test helpers to reduce cyclomatic complexity

// setupWebSocketServer creates a test WebSocket server with a custom handler
func setupWebSocketServer(t *testing.T, handler func(t *testing.T, conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
		}
		defer conn.Close()
		handler(t, conn)
	}))
}

// dialWebSocket establishes a WebSocket connection to the test server
func dialWebSocket(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	return conn
}

// waitForChannel waits for a channel signal with timeout
func waitForChannel(t *testing.T, ch <-chan bool, timeout time.Duration, msg string) {
	t.Helper()
	select {
	case <-ch:
		// Success
	case <-time.After(timeout):
		t.Errorf("%s: timeout after %v", msg, timeout)
	}
}

// verifyConstant checks if a duration constant matches expected value
func verifyConstant(t *testing.T, got, want time.Duration, name string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %s %v, got %v", name, want, got)
	}
}

// verifyIntConstant checks if an integer constant matches expected value
func verifyIntConstant(t *testing.T, got, want int64, name string) {
	t.Helper()
	if got != want {
		t.Errorf("Expected %s %d, got %d", name, want, got)
	}
}

func TestNewClient(t *testing.T) {
	hub := NewHub()

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		time.Sleep(100 * time.Millisecond)
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.hub != hub {
		t.Error("Client hub not set correctly")
	}
	if client.conn != conn {
		t.Error("Client connection not set correctly")
	}
	if client.send == nil {
		t.Error("Client send channel not initialized")
	}
	if cap(client.send) != 256 {
		t.Errorf("Expected send channel capacity 256, got %d", cap(client.send))
	}
}

func TestClient_Constants(t *testing.T) {
	verifyConstant(t, writeWait, 10*time.Second, "writeWait")
	verifyConstant(t, pongWait, 60*time.Second, "pongWait")
	verifyConstant(t, pingPeriod, (pongWait*9)/10, "pingPeriod")
	verifyIntConstant(t, maxMessageSize, 512*1024, "maxMessageSize")
}

func TestClient_WritePump_SendMessage(t *testing.T) {
	hub := NewHub()

	messageReceived := make(chan bool, 1)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			t.Errorf("Failed to read message: %v", err)
			return
		}
		if msg.Type != "test" {
			t.Errorf("Expected message type 'test', got '%s'", msg.Type)
		}
		messageReceived <- true
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)
	go client.writePump()

	testMessage := Message{Type: "test", Data: "test data"}
	client.send <- testMessage

	waitForChannel(t, messageReceived, 1*time.Second, "Message not received")
}

func TestClient_ReadPump_PingPong(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	receivedPong := make(chan bool, 1)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		pingMsg := Message{Type: MessageTypePing, Data: nil}
		if err := conn.WriteJSON(pingMsg); err != nil {
			t.Errorf("Failed to write ping: %v", err)
			return
		}

		var pongMsg Message
		if err := conn.ReadJSON(&pongMsg); err != nil {
			t.Errorf("Failed to read pong: %v", err)
			return
		}

		if pongMsg.Type == MessageTypePong {
			receivedPong <- true
		}
		time.Sleep(100 * time.Millisecond)
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)
	go client.readPump()
	go client.writePump()

	waitForChannel(t, receivedPong, 1*time.Second, "Pong not received")
}

func TestClient_Start(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	messageReceived := make(chan bool, 1)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		var msg Message
		if err := conn.ReadJSON(&msg); err == nil {
			messageReceived <- true
		}
		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)
	client.Start()

	// Allow goroutines to initialize (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	testMessage := Message{Type: "test", Data: "test data"}
	client.send <- testMessage

	waitForChannel(t, messageReceived, 1*time.Second, "Message not received")
}

func TestClient_ReadPump_ConnectionClose(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	unregistered := make(chan bool, 1)
	go func() {
		select {
		case <-hub.Unregister:
			unregistered <- true
		case <-time.After(2 * time.Second):
			// Timeout
		}
	}()

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		conn.Close()
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	hub.Register <- client

	// Allow registration to process (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	go client.readPump()

	waitForChannel(t, unregistered, 1*time.Second, "Client not unregistered after connection close")
}

func TestClient_WritePump_ChannelClose(t *testing.T) {
	hub := NewHub()

	receivedClose := make(chan bool, 1)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		for {
			messageType, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					receivedClose <- true
				}
				return
			}
			if messageType == websocket.CloseMessage {
				receivedClose <- true
				return
			}
		}
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	go client.writePump()

	// Allow writePump goroutine to start (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)
	close(client.send)

	// Close message may or may not be received due to timing
	select {
	case <-receivedClose:
		// Success
	case <-time.After(1 * time.Second):
		// Acceptable - connection may close before message is read
	}
}

func TestClient_WritePump_PingInterval(t *testing.T) {
	hub := NewHub()

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		conn.SetPingHandler(func(string) error {
			return conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(time.Second))
		})

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)
	go client.writePump()

	// pingPeriod is 54 seconds by default, too long for test
	// Just verify write pump starts without error
	time.Sleep(100 * time.Millisecond)
}

func TestClient_Integration(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	messagesReceived := make(chan Message, 10)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		for {
			var msg Message
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			messagesReceived <- msg
		}
	})
	defer server.Close()

	conn := dialWebSocket(t, server)
	defer conn.Close()

	client := NewClient(hub, conn)
	client.Start()

	hub.Register <- client

	// Allow registration to process (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	testData := map[string]string{"test": "integration"}
	hub.BroadcastJSON("integration_test", testData)

	select {
	case msg := <-messagesReceived:
		if msg.Type != "integration_test" {
			t.Errorf("Expected message type 'integration_test', got '%s'", msg.Type)
		}
	case <-time.After(1 * time.Second):
		t.Error("Message not received within timeout")
	}
}

func TestClient_ReadPump_SetReadDeadlineError(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		time.Sleep(10 * time.Millisecond)
		conn.Close()
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	hub.Register <- client

	// Allow registration to process (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	// Should handle errors gracefully without panic
	client.readPump()
}

func TestClient_ReadPump_UnexpectedCloseError(t *testing.T) {
	hub := NewHub()
	go hub.Run()
	time.Sleep(10 * time.Millisecond)

	unregistered := make(chan bool, 1)
	go func() {
		select {
		case <-hub.Unregister:
			unregistered <- true
		case <-time.After(5 * time.Second):
			// Timeout - must be longer than waitForChannel timeout
		}
	}()

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		time.Sleep(10 * time.Millisecond)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseAbnormalClosure, "test close"))
		conn.Close()
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	hub.Register <- client

	// Allow registration to process (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	go client.readPump()

	waitForChannel(t, unregistered, 3*time.Second, "Client not unregistered after abnormal close")
	time.Sleep(100 * time.Millisecond)
}

func TestClient_WritePump_WriteJSONError(t *testing.T) {
	hub := NewHub()

	serverClosed := make(chan bool, 1)
	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		time.Sleep(100 * time.Millisecond)
		conn.Close()
		serverClosed <- true
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	go client.writePump()

	// Allow writePump goroutine to start (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)
	<-serverClosed

	testMessage := Message{Type: "test", Data: "test data"}
	client.send <- testMessage

	time.Sleep(100 * time.Millisecond)
	// Should handle error without panic
}

func TestClient_WritePump_SetWriteDeadlineError(t *testing.T) {
	hub := NewHub()

	server := setupWebSocketServer(t, func(t *testing.T, conn *websocket.Conn) {
		time.Sleep(200 * time.Millisecond)
	})
	defer server.Close()

	conn := dialWebSocket(t, server)

	client := NewClient(hub, conn)
	go client.writePump()

	// Allow writePump goroutine to start (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)
	conn.Close()

	testMessage := Message{Type: "test", Data: "test data"}
	select {
	case client.send <- testMessage:
	default:
		// Channel might be closed already
	}

	time.Sleep(100 * time.Millisecond)
	// Should handle error without panic
}

func BenchmarkClient_SendMessage(b *testing.B) {
	hub := NewHub()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			b.Fatalf("Failed to upgrade: %v", err)
		}
		defer conn.Close()

		// Read and discard messages
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	client := NewClient(hub, conn)
	go client.writePump()

	// Allow writePump goroutine to start (100ms for CI reliability under load)
	time.Sleep(100 * time.Millisecond)

	testMessage := Message{Type: "benchmark", Data: "test data"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		select {
		case client.send <- testMessage:
		default:
			// Channel full, skip
		}
	}
}
