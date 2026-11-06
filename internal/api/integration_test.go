package api

import (
	"bytes"
	"encoding/base64"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/gorilla/websocket"
)

func TestAuthenticationFlowWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("TOMCHAT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TOMCHAT_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	defer adminPool.Close()

	schema := "tomchat_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	}()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	testPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect to isolated test schema: %v", err)
	}
	defer testPool.Close()

	handler := New(testPool, true)
	if err := handler.Migrate(); err != nil {
		t.Fatalf("run schema migrations: %v", err)
	}
	server := handler.Handler()

	registerBody := []byte(`{"username":"tomtest","password":"a-long-test-password"}`)
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	server.ServeHTTP(registerResponse, registerRequest)
	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}

	loginBody := []byte(`{"username":"tomtest","password":"a-long-test-password"}`)
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}
	var session authResponse
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if session.Token == "" || session.User.Username != "tomtest" {
		t.Fatalf("unexpected login response: %+v", session)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+session.Token)
	meResponse := httptest.NewRecorder()
	server.ServeHTTP(meResponse, meRequest)
	if meResponse.Code != http.StatusOK {
		t.Fatalf("current user: expected 200, got %d", meResponse.Code)
	}

	logoutRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+session.Token)
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d", logoutResponse.Code)
	}

	reuseRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	reuseRequest.Header.Set("Authorization", "Bearer "+session.Token)
	reuseResponse := httptest.NewRecorder()
	server.ServeHTTP(reuseResponse, reuseRequest)
	if reuseResponse.Code != http.StatusUnauthorized {
		t.Fatalf("revoked session: expected 401, got %d", reuseResponse.Code)
	}
}


func TestConversationPersistenceAndWebSocketDelivery(t *testing.T) {
	databaseURL := os.Getenv("TOMCHAT_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TOMCHAT_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test PostgreSQL: %v", err)
	}
	defer adminPool.Close()
	schema := "tomchat_messages_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("create isolated test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	}()
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("connect to isolated schema: %v", err)
	}
	defer pool.Close()

	api := New(pool, true)
	if err := api.Migrate(); err != nil {
		t.Fatalf("migrate schema: %v", err)
	}
	httpServer := httptest.NewServer(api.Handler())
	defer httpServer.Close()

	createAccount := func(username string) (user, string) {
		t.Helper()
		body, _ := json.Marshal(credentials{Username: username, Password: "long-test-password-123"})
		registerRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/auth/register", bytes.NewReader(body))
		registerRequest.Header.Set("Content-Type", "application/json")
		registerResponse, err := httpServer.Client().Do(registerRequest)
		if err != nil {
			t.Fatalf("register %s: %v", username, err)
		}
		defer registerResponse.Body.Close()
		if registerResponse.StatusCode != http.StatusCreated {
			t.Fatalf("register %s: got %d", username, registerResponse.StatusCode)
		}
		var account user
		if err := json.NewDecoder(registerResponse.Body).Decode(&account); err != nil {
			t.Fatalf("decode registered user: %v", err)
		}

		loginBody, _ := json.Marshal(credentials{Username: username, Password: "long-test-password-123"})
		loginRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/auth/login", bytes.NewReader(loginBody))
		loginRequest.Header.Set("Content-Type", "application/json")
		loginResponse, err := httpServer.Client().Do(loginRequest)
		if err != nil {
			t.Fatalf("login %s: %v", username, err)
		}
		defer loginResponse.Body.Close()
		if loginResponse.StatusCode != http.StatusOK {
			t.Fatalf("login %s: got %d", username, loginResponse.StatusCode)
		}
		var session authResponse
		if err := json.NewDecoder(loginResponse.Body).Decode(&session); err != nil {
			t.Fatalf("decode login response: %v", err)
		}
		return account, session.Token
	}

	alice, aliceToken := createAccount("alicechat")
	bob, bobToken := createAccount("bobchat")
	_, charlieToken := createAccount("charliechat")

	createBody, _ := json.Marshal(createConversationRequest{Kind: "direct", MemberIDs: []string{bob.ID}})
	createRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/conversations", bytes.NewReader(createBody))
	createRequest.Header.Set("Authorization", "Bearer "+aliceToken)
	createRequest.Header.Set("Content-Type", "application/json")
	createResponse, err := httpServer.Client().Do(createRequest)
	if err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	defer createResponse.Body.Close()
	if createResponse.StatusCode != http.StatusCreated {
		t.Fatalf("create conversation: got %d", createResponse.StatusCode)
	}
	var created conversation
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode conversation: %v", err)
	}
	if created.MemberCount != 2 || created.Kind != "direct" || len(created.MemberIDs) != 2 {
		t.Fatalf("unexpected conversation: %+v", created)
	}

	conversationListRequest, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/v1/conversations", nil)
	conversationListRequest.Header.Set("Authorization", "Bearer "+aliceToken)
	conversationListResponse, err := httpServer.Client().Do(conversationListRequest)
	if err != nil {
		t.Fatalf("list conversations: %v", err)
	}
	var conversationList struct {
		Conversations []conversation
	}
	if err := json.NewDecoder(conversationListResponse.Body).Decode(&conversationList); err != nil {
		t.Fatalf("decode conversation list: %v", err)
	}
	conversationListResponse.Body.Close()
	if len(conversationList.Conversations) != 1 || len(conversationList.Conversations[0].MemberIDs) != 2 {
		t.Fatalf("conversation list did not include members: %+v", conversationList)
	}

	unauthorizedHistoryRequest, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/v1/conversations/"+created.ID+"/messages", nil)
	unauthorizedHistoryRequest.Header.Set("Authorization", "Bearer "+charlieToken)
	unauthorizedHistoryResponse, err := httpServer.Client().Do(unauthorizedHistoryRequest)
	if err != nil {
		t.Fatalf("request history as non-member: %v", err)
	}
	unauthorizedHistoryResponse.Body.Close()
	if unauthorizedHistoryResponse.StatusCode != http.StatusForbidden {
		t.Fatalf("non-member history access: expected 403, got %d", unauthorizedHistoryResponse.StatusCode)
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+bobToken)
	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/v1/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("connect websocket: %v", err)
	}
	defer conn.Close()

	clientMessageID := uuid.NewString()
	sendBody, _ := json.Marshal(messageSendPayload{
		ConversationID: created.ID,
		ClientMessageID: clientMessageID,
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("opaque-ciphertext")),
		Nonce: base64.StdEncoding.EncodeToString([]byte("123456789012")),
		EncryptionVersion: 1,
		KeyID: "alice-device-key-1",
	})
	sendRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/conversations/"+created.ID+"/messages", bytes.NewReader(sendBody))
	sendRequest.Header.Set("Authorization", "Bearer "+aliceToken)
	sendRequest.Header.Set("Content-Type", "application/json")
	sendResponse, err := httpServer.Client().Do(sendRequest)
	if err != nil {
		t.Fatalf("send encrypted message: %v", err)
	}
	if sendResponse.StatusCode != http.StatusCreated {
		t.Fatalf("send encrypted message: got %d", sendResponse.StatusCode)
	}
	var sent storedMessage
	if err := json.NewDecoder(sendResponse.Body).Decode(&sent); err != nil {
		t.Fatalf("decode sent message: %v", err)
	}
	sendResponse.Body.Close()
	if sent.SenderID != alice.ID || sent.Ciphertext != base64.StdEncoding.EncodeToString([]byte("opaque-ciphertext")) {
		t.Fatalf("unexpected stored message: %+v", sent)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event wsEnvelope
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read realtime delivery: %v", err)
	}
	if event.Event != "message.deliver" || event.Version != 1 {
		t.Fatalf("unexpected websocket event: %+v", event)
	}
	var delivered storedMessage
	if err := json.Unmarshal(event.Payload, &delivered); err != nil {
		t.Fatalf("decode delivered message: %v", err)
	}
	if delivered.ID != sent.ID || delivered.ConversationID != created.ID {
		t.Fatalf("delivered message does not match persisted message: %+v", delivered)
	}

	historyRequest, _ := http.NewRequest(http.MethodGet, httpServer.URL+"/api/v1/conversations/"+created.ID+"/messages?limit=20", nil)
	historyRequest.Header.Set("Authorization", "Bearer "+bobToken)
	historyResponse, err := httpServer.Client().Do(historyRequest)
	if err != nil {
		t.Fatalf("load message history: %v", err)
	}
	defer historyResponse.Body.Close()
	if historyResponse.StatusCode != http.StatusOK {
		t.Fatalf("load message history: got %d", historyResponse.StatusCode)
	}
	var history messagePage
	if err := json.NewDecoder(historyResponse.Body).Decode(&history); err != nil {
		t.Fatalf("decode message history: %v", err)
	}
	if len(history.Messages) != 1 || history.Messages[0].ID != sent.ID {
		t.Fatalf("message history missing persisted message: %+v", history)
	}

	// Retrying the same client-generated ID is idempotent and does not duplicate rows.
	retryRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/conversations/"+created.ID+"/messages", bytes.NewReader(sendBody))
	retryRequest.Header.Set("Authorization", "Bearer "+aliceToken)
	retryRequest.Header.Set("Content-Type", "application/json")
	retryResponse, err := httpServer.Client().Do(retryRequest)
	if err != nil {
		t.Fatalf("retry message: %v", err)
	}
	defer retryResponse.Body.Close()
	if retryResponse.StatusCode != http.StatusOK {
		t.Fatalf("idempotent retry: expected 200, got %d", retryResponse.StatusCode)
	}
	var retried storedMessage
	if err := json.NewDecoder(retryResponse.Body).Decode(&retried); err != nil {
		t.Fatalf("decode retried message: %v", err)
	}
	if retried.ID != sent.ID {
		t.Fatalf("idempotent retry returned %s, want %s", retried.ID, sent.ID)
	}
	var changedPayload messageSendPayload
	if err := json.Unmarshal(sendBody, &changedPayload); err != nil {
		t.Fatalf("decode retry payload: %v", err)
	}
	changedPayload.Ciphertext = base64.StdEncoding.EncodeToString([]byte("different-ciphertext"))
	changedBody, _ := json.Marshal(changedPayload)
	conflictRequest, _ := http.NewRequest(http.MethodPost, httpServer.URL+"/api/v1/conversations/"+created.ID+"/messages", bytes.NewReader(changedBody))
	conflictRequest.Header.Set("Authorization", "Bearer "+aliceToken)
	conflictRequest.Header.Set("Content-Type", "application/json")
	conflictResponse, err := httpServer.Client().Do(conflictRequest)
	if err != nil {
		t.Fatalf("submit reused message ID with changed payload: %v", err)
	}
	conflictResponse.Body.Close()
	if conflictResponse.StatusCode != http.StatusConflict {
		t.Fatalf("changed payload with reused message ID: expected 409, got %d", conflictResponse.StatusCode)
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE conversation_id = $1", uuid.MustParse(created.ID)).Scan(&count); err != nil {
		t.Fatalf("count persisted messages: %v", err)
	}
	if count != 1 {
		t.Fatalf("stored %d messages after retry; want exactly one", count)
	}

	aliceHeaders := http.Header{}
	aliceHeaders.Set("Authorization", "Bearer "+aliceToken)
	aliceWS, _, err := websocket.DefaultDialer.Dial(wsURL, aliceHeaders)
	if err != nil {
		t.Fatalf("connect sender websocket: %v", err)
	}
	defer aliceWS.Close()

	secondPayload, _ := json.Marshal(messageSendPayload{
		ConversationID: created.ID,
		ClientMessageID: uuid.NewString(),
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("second-opaque-ciphertext")),
		Nonce: base64.StdEncoding.EncodeToString([]byte("abcdefghijkl")),
		EncryptionVersion: 1,
	})
	requestID := uuid.NewString()
	if err := aliceWS.WriteJSON(wsEnvelope{
		Version: 1, Event: "message.send", RequestID: requestID, Payload: secondPayload,
	}); err != nil {
		t.Fatalf("send websocket message: %v", err)
	}

	_ = aliceWS.SetReadDeadline(time.Now().Add(3 * time.Second))
	var senderDelivery wsEnvelope
	if err := aliceWS.ReadJSON(&senderDelivery); err != nil {
		t.Fatalf("read sender delivery event: %v", err)
	}
	if senderDelivery.Event != "message.deliver" {
		t.Fatalf("sender expected message.deliver first, got %s", senderDelivery.Event)
	}
	var senderAck wsEnvelope
	if err := aliceWS.ReadJSON(&senderAck); err != nil {
		t.Fatalf("read sender acknowledgement: %v", err)
	}
	if senderAck.Event != "message.ack" || senderAck.RequestID != requestID {
		t.Fatalf("unexpected sender acknowledgement: %+v", senderAck)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var secondDelivery wsEnvelope
	if err := conn.ReadJSON(&secondDelivery); err != nil {
		t.Fatalf("read second websocket delivery: %v", err)
	}
	var secondMessage storedMessage
	if err := json.Unmarshal(secondDelivery.Payload, &secondMessage); err != nil {
		t.Fatalf("decode second websocket delivery: %v", err)
	}
	if secondDelivery.Event != "message.deliver" || secondMessage.Ciphertext != base64.StdEncoding.EncodeToString([]byte("second-opaque-ciphertext")) {
		t.Fatalf("unexpected second websocket message: %+v", secondMessage)
	}
	if err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM messages WHERE conversation_id = $1", created.ID).Scan(&count); err != nil {
		t.Fatalf("count websocket-persisted messages: %v", err)
	}
	if count != 2 {
		t.Fatalf("stored %d messages after websocket send; want 2", count)
	}
}
