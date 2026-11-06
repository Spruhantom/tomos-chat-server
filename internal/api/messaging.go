package api

import (
	"context"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
)

var (
	errNotConversationMember = errors.New("not a conversation member")
	errInvalidMessage = errors.New("invalid encrypted message")
	errMessageIDConflict = errors.New("client message id already used")
)

const (
	maxCiphertextBytes = 40 * 1024
	maxWebSocketFrameBytes = 64 * 1024
)

type conversation struct {
	ID string `json:"id"`
	Kind string `json:"kind"`
	Title *string `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	MemberCount int `json:"member_count"`
	MemberIDs []string `json:"member_ids"`
}

type createConversationRequest struct {
	Kind string `json:"kind"`
	Title string `json:"title"`
	MemberIDs []string `json:"member_ids"`
}

type messageSendPayload struct {
	ConversationID string `json:"conversation_id"`
	ClientMessageID string `json:"client_message_id"`
	Ciphertext string `json:"ciphertext"`
	Nonce string `json:"nonce"`
	EncryptionVersion int `json:"encryption_version"`
	KeyID string `json:"key_id,omitempty"`
}

type storedMessage struct {
	ID string `json:"id"`
	Sequence int64 `json:"sequence"`
	ConversationID string `json:"conversation_id"`
	SenderID string `json:"sender_id"`
	ClientMessageID string `json:"client_message_id"`
	Ciphertext string `json:"ciphertext"`
	Nonce string `json:"nonce"`
	EncryptionVersion int `json:"encryption_version"`
	KeyID string `json:"key_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type messagePage struct {
	Messages []storedMessage `json:"messages"`
	NextBefore int64 `json:"next_before,omitempty"`
}

type wsEnvelope struct {
	Version int `json:"version"`
	Event string `json:"event"`
	RequestID string `json:"request_id"`
	Payload json.RawMessage `json:"payload"`
}

type wsHub struct {
	mu sync.RWMutex
	users map[string]map[*wsClient]struct{}
}

type sessionContextKey struct{}

type wsClient struct {
	conn *websocket.Conn
	send chan []byte
	done chan struct{}
	once sync.Once
	userID string
	sessionHash []byte
}

func newWSHub() *wsHub {
	return &wsHub{users: make(map[string]map[*wsClient]struct{})}
}

func (h *wsHub) register(client *wsClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.users[client.userID]) >= 5 {
		return false
	}
	if h.users[client.userID] == nil {
		h.users[client.userID] = make(map[*wsClient]struct{})
	}
	h.users[client.userID][client] = struct{}{}
	return true
}

func (h *wsHub) unregister(client *wsClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients := h.users[client.userID]; clients != nil {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.users, client.userID)
		}
	}
}

func (h *wsHub) closeSession(sessionHash []byte) {
	h.mu.RLock()
	clients := make([]*wsClient, 0)
	for _, userClients := range h.users {
		for client := range userClients {
			if bytes.Equal(client.sessionHash, sessionHash) {
				clients = append(clients, client)
			}
		}
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.close()
	}
}

func (h *wsHub) broadcast(userIDs []string, payload []byte) {
	h.mu.RLock()
	clients := make([]*wsClient, 0)
	for _, userID := range userIDs {
		for client := range h.users[userID] {
			clients = append(clients, client)
		}
	}
	h.mu.RUnlock()
	for _, client := range clients {
		select {
		case <-client.done:
		case client.send <- payload:
		default:
			client.close()
		}
	}
}

func (c *wsClient) close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (a *API) createConversation(w http.ResponseWriter, r *http.Request) {
	current, ok := userFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	var input createConversationRequest
	if !decodeJSON(w, r, &input) {
		return
	}
	input.Kind = strings.TrimSpace(input.Kind)
	input.Title = strings.TrimSpace(input.Title)
	if input.Kind != "direct" && input.Kind != "group" {
		writeError(w, http.StatusBadRequest, "invalid_conversation_kind")
		return
	}
	if len(input.Title) > 120 {
		writeError(w, http.StatusBadRequest, "conversation_title_too_long")
		return
	}
	currentID, err := uuid.Parse(current.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	members := map[uuid.UUID]struct{}{currentID: {}}
	for _, rawID := range input.MemberIDs {
		id, parseErr := uuid.Parse(rawID)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid_member_id")
			return
		}
		members[id] = struct{}{}
	}
	if input.Kind == "direct" && len(members) != 2 {
		writeError(w, http.StatusBadRequest, "direct_conversation_requires_two_members")
		return
	}
	if input.Kind == "group" && (len(members) < 2 || len(members) > 100) {
		writeError(w, http.StatusBadRequest, "group_conversation_requires_2_to_100_members")
		return
	}
	memberIDs := make([]uuid.UUID, 0, len(members))
	for id := range members {
		memberIDs = append(memberIDs, id)
	}
	var existing int
	if err := a.db.QueryRow(r.Context(), "SELECT COUNT(*) FROM users WHERE id = ANY($1::uuid[])", memberIDs).Scan(&existing); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	if existing != len(memberIDs) {
		writeError(w, http.StatusBadRequest, "unknown_conversation_member")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	tx, err := a.db.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	defer tx.Rollback(ctx)
	id := uuid.New()
	var title any
	if input.Title != "" {
		title = input.Title
	}
	var createdAt time.Time
	if err := tx.QueryRow(ctx, "INSERT INTO conversations (id, kind, title, created_by) VALUES ($1, $2, $3, $4) RETURNING created_at", id, input.Kind, title, currentID).Scan(&createdAt); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	for _, memberID := range memberIDs {
		if _, err := tx.Exec(ctx, "INSERT INTO conversation_members (conversation_id, user_id) VALUES ($1, $2)", id, memberID); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	var responseTitle *string
	if input.Title != "" {
		responseTitle = &input.Title
	}
	memberIDStrings := make([]string, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		memberIDStrings = append(memberIDStrings, memberID.String())
	}
	writeJSON(w, http.StatusCreated, conversation{ID: id.String(), Kind: input.Kind, Title: responseTitle, CreatedAt: createdAt, MemberCount: len(memberIDs), MemberIDs: memberIDStrings})
}

func (a *API) listConversations(w http.ResponseWriter, r *http.Request) {
	current, ok := userFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	query := "SELECT c.id::text, c.kind, c.title, c.created_at, MAX(m.created_at), COUNT(DISTINCT all_members.user_id), " +
		"ARRAY_AGG(DISTINCT all_members.user_id::text) " +
		"FROM conversations c JOIN conversation_members mine ON mine.conversation_id = c.id AND mine.user_id = $1 " +
		"JOIN conversation_members all_members ON all_members.conversation_id = c.id " +
		"LEFT JOIN messages m ON m.conversation_id = c.id GROUP BY c.id " +
		"ORDER BY COALESCE(MAX(m.created_at), c.created_at) DESC, c.id DESC"
	currentID, err := uuid.Parse(current.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	rows, err := a.db.Query(r.Context(), query, currentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	defer rows.Close()
	items := make([]conversation, 0)
	for rows.Next() {
		var item conversation
		if err := rows.Scan(&item.ID, &item.Kind, &item.Title, &item.CreatedAt, &item.LastMessageAt, &item.MemberCount, &item.MemberIDs); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": items})
}

func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	current, ok := userFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	conversationID, err := uuid.Parse(r.PathValue("conversationID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_conversation_id")
		return
	}
	if !a.isConversationMember(r.Context(), conversationID, current.ID) {
		writeError(w, http.StatusForbidden, "not_a_conversation_member")
		return
	}
	limit := int64(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || limit < 1 || limit > 100 {
			writeError(w, http.StatusBadRequest, "invalid_message_limit")
			return
		}
	}
	var before any
	if raw := r.URL.Query().Get("before"); raw != "" {
		value, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || value < 1 {
			writeError(w, http.StatusBadRequest, "invalid_message_cursor")
			return
		}
		before = value
	}
	query := "SELECT id::text, sequence, conversation_id::text, sender_id::text, client_message_id::text, " +
		"ciphertext, nonce, encryption_version, COALESCE(key_id, ''), created_at " +
		"FROM messages WHERE conversation_id = $1 AND sequence < COALESCE($2::bigint, 9223372036854775807) " +
		"ORDER BY sequence DESC LIMIT $3"
	rows, err := a.db.Query(r.Context(), query, conversationID, before, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	defer rows.Close()
	items := make([]storedMessage, 0, limit)
	for rows.Next() {
		var item storedMessage
		var ciphertext, nonce []byte
		if err := rows.Scan(&item.ID, &item.Sequence, &item.ConversationID, &item.SenderID, &item.ClientMessageID, &ciphertext, &nonce, &item.EncryptionVersion, &item.KeyID, &item.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		item.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
		item.Nonce = base64.StdEncoding.EncodeToString(nonce)
		items = append(items, item)
	}
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "internal_error")
		return
	}
	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}
	page := messagePage{Messages: items}
	if len(items) == int(limit) {
		page.NextBefore = items[0].Sequence
	}
	writeJSON(w, http.StatusOK, page)
}

func (a *API) sendMessageHTTP(w http.ResponseWriter, r *http.Request) {
	current, ok := userFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	var input messageSendPayload
	if !decodeJSON(w, r, &input) {
		return
	}
	if input.ConversationID != r.PathValue("conversationID") {
		writeError(w, http.StatusBadRequest, "conversation_id_mismatch")
		return
	}
	message, err := a.persistMessage(r.Context(), current, input)
	if err != nil {
		a.writeMessageError(w, err)
		return
	}
	a.publishMessage(r.Context(), message)
	writeJSON(w, http.StatusCreated, message)
}

func (a *API) persistMessage(ctx context.Context, sender user, input messageSendPayload) (storedMessage, error) {
	conversationID, err := uuid.Parse(input.ConversationID)
	if err != nil {
		return storedMessage{}, errInvalidMessage
	}
	senderID, err := uuid.Parse(sender.ID)
	if err != nil {
		return storedMessage{}, err
	}
	clientMessageID, err := uuid.Parse(input.ClientMessageID)
	if err != nil || input.EncryptionVersion != 1 || len(input.KeyID) > 128 {
		return storedMessage{}, errInvalidMessage
	}
	ciphertext, err := base64.StdEncoding.DecodeString(input.Ciphertext)
	if err != nil || len(ciphertext) == 0 || len(ciphertext) > maxCiphertextBytes {
		return storedMessage{}, errInvalidMessage
	}
	nonce, err := base64.StdEncoding.DecodeString(input.Nonce)
	if err != nil || len(nonce) < 8 || len(nonce) > 32 {
		return storedMessage{}, errInvalidMessage
	}
	if !a.isConversationMember(ctx, conversationID, senderID.String()) {
		return storedMessage{}, errNotConversationMember
	}
	var item storedMessage
	insertQuery := "INSERT INTO messages (id, conversation_id, sender_id, client_message_id, ciphertext, nonce, encryption_version, key_id) " +
		"VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')) ON CONFLICT (sender_id, client_message_id) DO NOTHING " +
		"RETURNING id::text, sequence, conversation_id::text, sender_id::text, client_message_id::text, " +
		"ciphertext, nonce, encryption_version, COALESCE(key_id, ''), created_at"
	item, err = scanStoredMessage(a.db.QueryRow(ctx, insertQuery, uuid.New(), conversationID, senderID, clientMessageID, ciphertext, nonce, input.EncryptionVersion, input.KeyID))
	if errors.Is(err, pgx.ErrNoRows) {
		selectQuery := "SELECT id::text, sequence, conversation_id::text, sender_id::text, client_message_id::text, " +
			"ciphertext, nonce, encryption_version, COALESCE(key_id, ''), created_at " +
			"FROM messages WHERE sender_id = $1 AND client_message_id = $2"
		item, err = scanStoredMessage(a.db.QueryRow(ctx, selectQuery, sender.ID, clientMessageID))
		if err != nil {
			return storedMessage{}, err
		}
		if item.ConversationID != conversationID.String() {
			return storedMessage{}, errMessageIDConflict
		}
		return item, nil
	}
	if err != nil {
		return storedMessage{}, err
	}
	return item, nil
}

func scanStoredMessage(row pgx.Row) (storedMessage, error) {
	var item storedMessage
	var ciphertext, nonce []byte
	err := row.Scan(&item.ID, &item.Sequence, &item.ConversationID, &item.SenderID, &item.ClientMessageID, &ciphertext, &nonce, &item.EncryptionVersion, &item.KeyID, &item.CreatedAt)
	if err != nil {
		return storedMessage{}, err
	}
	item.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	item.Nonce = base64.StdEncoding.EncodeToString(nonce)
	return item, nil
}

func (a *API) publishMessage(ctx context.Context, item storedMessage) {
	conversationID, parseErr := uuid.Parse(item.ConversationID)
	if parseErr != nil {
		log.Printf("invalid conversation ID while publishing message: %v", parseErr)
		return
	}
	rows, err := a.db.Query(ctx, "SELECT user_id::text FROM conversation_members WHERE conversation_id = $1", conversationID)
	if err != nil {
		log.Printf("load message recipients: %v", err)
		return
	}
	defer rows.Close()
	recipients := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Printf("scan message recipient: %v", err)
			return
		}
		recipients = append(recipients, id)
	}
	if rows.Err() != nil {
		log.Printf("iterate message recipients: %v", rows.Err())
		return
	}
	payload, err := json.Marshal(wsEnvelope{Version: 1, Event: "message.deliver", RequestID: uuid.NewString(), Payload: mustJSON(item)})
	if err != nil {
		log.Printf("encode message event: %v", err)
		return
	}
	a.hub.broadcast(recipients, payload)
}

func mustJSON(value any) json.RawMessage {
	encoded, _ := json.Marshal(value)
	return encoded
}

func (a *API) writeMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errInvalidMessage):
		writeError(w, http.StatusBadRequest, "invalid_encrypted_message")
	case errors.Is(err, errNotConversationMember):
		writeError(w, http.StatusForbidden, "not_a_conversation_member")
	case errors.Is(err, errMessageIDConflict):
		writeError(w, http.StatusConflict, "client_message_id_conflict")
	default:
		log.Printf("message operation: %v", err)
		writeError(w, http.StatusInternalServerError, "internal_error")
	}
}

func (a *API) isConversationMember(ctx context.Context, conversationID uuid.UUID, userID string) bool {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return false
	}
	var member bool
	err = a.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM conversation_members WHERE conversation_id = $1 AND user_id = $2)", conversationID, parsedUserID).Scan(&member)
	return err == nil && member
}

func (a *API) websocketHandler(w http.ResponseWriter, r *http.Request) {
	current, ok := userFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication_required")
		return
	}
	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	sessionHash, ok := r.Context().Value(sessionContextKey{}).([]byte)
	if !ok || len(sessionHash) == 0 {
		_ = conn.Close()
		return
	}
	client := &wsClient{conn: conn, send: make(chan []byte, 64), done: make(chan struct{}), userID: current.ID, sessionHash: append([]byte(nil), sessionHash...)}
	if !a.hub.register(client) {
		client.close()
		return
	}
	go client.writePump()
	client.readPump(a, current)
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	EnableCompression: false,
}

func (c *wsClient) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	defer c.conn.Close()
	for {
		select {
		case <-c.done:
			_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.close()
				return
			}
		}
	}
}

func (c *wsClient) readPump(a *API, current user) {
	defer a.hub.unregister(c)
	defer c.close()
	c.conn.SetReadLimit(maxWebSocketFrameBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		if !a.sessionIsActive(c.sessionHash) {
			return errors.New("session expired")
		}
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	windowStarted := time.Now()
	messageCount := 0
		for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if time.Since(windowStarted) >= time.Minute {
			windowStarted = time.Now()
			messageCount = 0
		}
		messageCount++
		if messageCount > 100 {
			c.sendError("", "rate_limit_exceeded")
			return
		}
		var envelope wsEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil || envelope.Version != 1 || envelope.RequestID == "" {
			c.sendError("", "invalid_event")
			continue
		}
		switch envelope.Event {
		case "message.send":
			var input messageSendPayload
			if err := json.Unmarshal(envelope.Payload, &input); err != nil {
				c.sendError(envelope.RequestID, "invalid_message")
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			item, err := a.persistMessage(ctx, current, input)
			if err == nil {
				a.publishMessage(ctx, item)
			}
			cancel()
			if err != nil {
				c.sendError(envelope.RequestID, messageErrorCode(err))
				continue
			}
			c.sendEvent(wsEnvelope{Version: 1, Event: "message.ack", RequestID: envelope.RequestID, Payload: mustJSON(map[string]any{"message_id": item.ID, "sequence": item.Sequence})})
		default:
			c.sendError(envelope.RequestID, "unsupported_event")
		}
	}
}

func (c *wsClient) sendEvent(event wsEnvelope) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	select {
	case <-c.done:
	case c.send <- payload:
	default:
		c.close()
	}
}

func (c *wsClient) sendError(requestID, code string) {
	c.sendEvent(wsEnvelope{Version: 1, Event: "error", RequestID: requestID, Payload: mustJSON(map[string]string{"code": code})})
}

func messageErrorCode(err error) string {
	switch {
	case errors.Is(err, errInvalidMessage):
		return "invalid_encrypted_message"
	case errors.Is(err, errNotConversationMember):
		return "not_a_conversation_member"
	case errors.Is(err, errMessageIDConflict):
		return "client_message_id_conflict"
	default:
		return "internal_error"
	}
}

func (a *API) sessionIsActive(sessionHash []byte) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var active bool
	err := a.db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM sessions WHERE token_hash = $1 AND expires_at > NOW())", sessionHash).Scan(&active)
	return err == nil && active
}
