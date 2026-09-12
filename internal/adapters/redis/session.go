package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	redisclient "github.com/redis/go-redis/v9"

	"github.com/wecratfs/commerce/internal/ports"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionStore struct {
	client *redisclient.Client
	prefix string
}

func NewSessionStore(client *redisclient.Client) *SessionStore {
	return &SessionStore{client: client, prefix: "wecratfs"}
}

func (s *SessionStore) Create(ctx context.Context, session ports.SessionRecord) error {
	if session.ID == "" || session.UserID <= 0 || !session.ExpiresAt.After(time.Now()) {
		return errors.New("invalid session record")
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return err
	}
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return errors.New("session already expired")
	}
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, s.sessionKey(session.ID), payload, ttl)
	pipe.SAdd(ctx, s.userKey(session.UserID), session.ID)
	pipe.Expire(ctx, s.userKey(session.UserID), ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SessionStore) Get(ctx context.Context, sessionID string) (ports.SessionRecord, error) {
	value, err := s.client.Get(ctx, s.sessionKey(sessionID)).Bytes()
	if errors.Is(err, redisclient.Nil) {
		return ports.SessionRecord{}, ErrSessionNotFound
	}
	if err != nil {
		return ports.SessionRecord{}, err
	}
	var session ports.SessionRecord
	if err := json.Unmarshal(value, &session); err != nil {
		return ports.SessionRecord{}, err
	}
	return session, nil
}

func (s *SessionStore) Delete(ctx context.Context, sessionID string) error {
	session, err := s.Get(ctx, sessionID)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, s.sessionKey(sessionID))
	if session.UserID > 0 {
		pipe.SRem(ctx, s.userKey(session.UserID), sessionID)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SessionStore) DeleteAll(ctx context.Context, userID int64) error {
	ids, err := s.sessionIDs(ctx, userID)
	if err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	for _, id := range ids {
		pipe.Del(ctx, s.sessionKey(id))
	}
	pipe.Del(ctx, s.userKey(userID))
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SessionStore) List(ctx context.Context, userID int64) ([]ports.SessionRecord, error) {
	ids, err := s.sessionIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]ports.SessionRecord, 0, len(ids))
	for _, id := range ids {
		session, err := s.Get(ctx, id)
		if errors.Is(err, ErrSessionNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		result = append(result, session)
	}
	return result, nil
}

// sessionIDs bounds each user-facing session operation. Expired Redis set
// members are disposable and should never turn a session request into an
// unbounded read or transaction.
func (s *SessionStore) sessionIDs(ctx context.Context, userID int64) ([]string, error) {
	ids := make([]string, 0, 16)
	var cursor uint64
	for {
		batch, next, err := s.client.SScan(ctx, s.userKey(userID), cursor, "", 128).Result()
		if err != nil {
			return nil, err
		}
		remaining := 128 - len(ids)
		if len(batch) > remaining {
			batch = batch[:remaining]
		}
		ids = append(ids, batch...)
		if len(ids) >= 128 || next == 0 {
			return ids, nil
		}
		cursor = next
	}
}

func (s *SessionStore) sessionKey(id string) string {
	return s.prefix + ":session:" + hash(id)
}

func (s *SessionStore) userKey(userID int64) string {
	return s.prefix + ":user-sessions:" + strconv.FormatInt(userID, 10)
}

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
