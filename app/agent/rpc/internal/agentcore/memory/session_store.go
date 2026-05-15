package memory

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"xls/app/agent/rpc/internal/model"
)

// MessageStore 是现有接口，兼容旧代码。
type MessageStore interface {
	Read(ctx context.Context, sessionID string) ([]*schema.Message, error)
	Write(ctx context.Context, sessionID string, messages []*schema.Message) error
}

// SessionStore 是业务层接口，对标 eino-examples 的 mem.Store。
type SessionStore interface {
	// GetOrCreate 获取或创建一个 Session。
	// 如果 sessionUUID 为空，则创建新 Session 并写入 db。
	GetOrCreate(ctx context.Context, userID uint64, sessionUUID string) (*Session, error)
	// Append 追加一条消息到 Session，并写入 db。
	Append(ctx context.Context, sessionID uint, msg *schema.Message) error
	// GetMessages 返回 Session 的所有历史消息（按创建时间顺序）。
	GetMessages(ctx context.Context, sessionID uint) ([]*schema.Message, error)
	// List 返回用户的所有 Session 摘要。
	List(ctx context.Context, userID uint64) ([]SessionMeta, error)
	// Delete 删除一个 Session 及其所有消息。
	Delete(ctx context.Context, sessionUUID string) error
	// UpdateTitle 更新 Session 标题。
	UpdateTitle(ctx context.Context, sessionID uint, title string) error
}

// SessionMeta 是 Session 的摘要信息，用于列表展示。
type SessionMeta struct {
	ID        uint      `json:"id"`
	UUID      string    `json:"uuid"`
	Title     string    `json:"title"`
	UserID    uint64    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Session 对标 eino-examples 的 mem.Session，
// 封装一个 MySQL Session 及其 in-memory 缓存。
type Session struct {
	ID     uint   `json:"id"`
	UUID   string `json:"uuid"`
	UserID uint64 `json:"user_id"`
	Title_ string `json:"title"` // 内部存储，避免与方法同名冲突

	createdAt  time.Time
	filePath   string
	mu         sync.Mutex
	messages   []*schema.Message
	dirty      bool
	maxHistory int
}

// Append 追加消息到内存缓冲，标记 dirty。
func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
	s.dirty = true
	return nil
}

// GetMessages 返回消息快照。
func (s *Session) GetMessages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Title 返回会话标题。
func (s *Session) Title() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Title_
}

// SessionStoreImpl 是基于 MySQL 的 SessionStore 实现。
type SessionStoreImpl struct {
	db         *gorm.DB
	maxHistory int
}

func NewSessionStoreImpl(db *gorm.DB, maxHistory int) *SessionStoreImpl {
	if maxHistory <= 0 {
		maxHistory = 50
	}
	return &SessionStoreImpl{
		db:         db,
		maxHistory: maxHistory,
	}
}

// GetOrCreate 获取或创建 Session。
// 如果 sessionUUID 为空，则创建新的。
func (s *SessionStoreImpl) GetOrCreate(ctx context.Context, userID uint64, sessionUUID string) (*Session, error) {
	var sess model.Session

	if sessionUUID != "" {
		err := s.db.WithContext(ctx).
			Where("uuid = ? AND user_id = ?", sessionUUID, userID).
			First(&sess).Error
		if err == nil {
			return &Session{
				ID:        sess.ID,
				UUID:      sess.UUID,
				UserID:    sess.UserID,
				Title_:    sess.Title,
				createdAt: sess.CreatedAt,
				messages:  nil,
				dirty:     false,
			}, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	// 创建新 Session
	if sessionUUID == "" {
		sessionUUID = uuid.New().String()
	}
	sess = model.Session{
		UUID:   sessionUUID,
		UserID: userID,
		Title:  "新会话",
	}
	if err := s.db.WithContext(ctx).Create(&sess).Error; err != nil {
		return nil, err
	}

	return &Session{
		ID:        sess.ID,
		UUID:      sess.UUID,
		UserID:    sess.UserID,
		Title_:    sess.Title,
		createdAt: sess.CreatedAt,
		messages:  nil,
		dirty:     false,
	}, nil
}

// Append 追加消息到 db。
func (s *SessionStoreImpl) Append(ctx context.Context, sessionID uint, msg *schema.Message) error {
	dbMsg := model.SessionMessage{
		SessionID: sessionID,
		Role:     string(msg.Role),
		Content:  msg.Content,
	}
	if err := s.db.WithContext(ctx).Create(&dbMsg).Error; err != nil {
		return err
	}

	// 如果是第一条用户消息，更新 Session 标题
	var count int64
	s.db.WithContext(ctx).Model(&model.SessionMessage{}).
		Where("session_id = ? AND role = ?", sessionID, schema.User).
		Count(&count)
	if count == 1 && msg.Role == schema.User {
		title := msg.Content
		if utf8.RuneCountInString(title) > 60 {
			title = string([]rune(title)[:60]) + "..."
		}
		_ = s.db.WithContext(ctx).Model(&model.Session{}).
			Where("id = ?", sessionID).
			Update("title", title).Error
	}

	// 限制历史消息数量（保留最近 N 条）
	var latestIDs []uint
	s.db.WithContext(ctx).Model(&model.SessionMessage{}).
		Where("session_id = ?", sessionID).
		Order("id DESC").
		Limit(s.maxHistory+10).
		Pluck("id", &latestIDs)

	if len(latestIDs) > s.maxHistory {
		idsToDelete := latestIDs[s.maxHistory:]
		s.db.WithContext(ctx).Where("id IN ?", idsToDelete).Delete(&model.SessionMessage{})
	}

	return nil
}

// GetMessages 读取 Session 的所有消息。
func (s *SessionStoreImpl) GetMessages(ctx context.Context, sessionID uint) ([]*schema.Message, error) {
	var dbMsgs []model.SessionMessage
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("id ASC").
		Find(&dbMsgs).Error; err != nil {
		return nil, err
	}

	messages := make([]*schema.Message, 0, len(dbMsgs))
	for _, m := range dbMsgs {
		messages = append(messages, &schema.Message{
			Role:    schema.RoleType(m.Role),
			Content: m.Content,
		})
	}
	return messages, nil
}

// List 返回用户所有 Session 摘要。
func (s *SessionStoreImpl) List(ctx context.Context, userID uint64) ([]SessionMeta, error) {
	var sessions []model.Session
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	metas := make([]SessionMeta, 0, len(sessions))
	for _, sess := range sessions {
		metas = append(metas, SessionMeta{
			ID:        sess.ID,
			UUID:      sess.UUID,
			Title:     sess.Title,
			UserID:    sess.UserID,
			CreatedAt: sess.CreatedAt,
		})
	}
	return metas, nil
}

// Delete 删除 Session 及其所有消息。
func (s *SessionStoreImpl) Delete(ctx context.Context, sessionUUID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sess model.Session
		if err := tx.Where("uuid = ?", sessionUUID).First(&sess).Error; err != nil {
			return err
		}
		if err := tx.Where("session_id = ?", sess.ID).Delete(&model.SessionMessage{}).Error; err != nil {
			return err
		}
		return tx.Delete(&sess).Error
	})
}

// UpdateTitle 更新 Session 标题。
func (s *SessionStoreImpl) UpdateTitle(ctx context.Context, sessionID uint, title string) error {
	return s.db.WithContext(ctx).Model(&model.Session{}).
		Where("id = ?", sessionID).
		Update("title", title).Error
}

// InMemorySessionStore 内存存储（原有实现）。
type InMemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string][]*schema.Message
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string][]*schema.Message),
	}
}

func (s *InMemorySessionStore) Read(ctx context.Context, sessionID string) ([]*schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	messages, ok := s.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	result := make([]*schema.Message, len(messages))
	copy(result, messages)
	return result, nil
}

func (s *InMemorySessionStore) Write(ctx context.Context, sessionID string, messages []*schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionID] = messages
	return nil
}

// InMemorySessionStoreAdapter 将 InMemorySessionStore 适配为 SessionStore。
type InMemorySessionStoreAdapter struct {
	inner *InMemorySessionStore
}

func NewInMemorySessionStoreAdapter() *InMemorySessionStoreAdapter {
	return &InMemorySessionStoreAdapter{
		inner: NewInMemorySessionStore(),
	}
}

func (a *InMemorySessionStoreAdapter) GetOrCreate(ctx context.Context, userID uint64, sessionUUID string) (*Session, error) {
	if sessionUUID == "" {
		sessionUUID = uuid.New().String()
	}
	return &Session{
		UUID:     sessionUUID,
		UserID:   userID,
		Title_:   "新会话",
		messages: nil,
	}, nil
}

func (a *InMemorySessionStoreAdapter) Append(ctx context.Context, sessionID uint, msg *schema.Message) error {
	return nil
}

func (a *InMemorySessionStoreAdapter) GetMessages(ctx context.Context, sessionID uint) ([]*schema.Message, error) {
	return nil, nil
}

func (a *InMemorySessionStoreAdapter) List(ctx context.Context, userID uint64) ([]SessionMeta, error) {
	return nil, nil
}

func (a *InMemorySessionStoreAdapter) Delete(ctx context.Context, sessionUUID string) error {
	return nil
}

func (a *InMemorySessionStoreAdapter) UpdateTitle(ctx context.Context, sessionID uint, title string) error {
	return nil
}

// MarshalJSON / UnmarshalJSON 用于兼容 proto 中的 Role 类型
func marshalRole(r schema.RoleType) string {
	b, _ := json.Marshal(r)
	return string(b)
}
