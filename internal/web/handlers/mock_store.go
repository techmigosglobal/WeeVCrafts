package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/wecratfs/commerce/internal/web/viewmodels"
)

type mockSession struct {
	Cart     []viewmodels.CartItem
	Wishlist map[string]bool
	Query    string
	Category string
	Sort     string
	Orders   []viewmodels.Order
}

type mockStore struct {
	mu       sync.Mutex
	sessions map[string]*mockSession
}

func newMockStore() *mockStore { return &mockStore{sessions: make(map[string]*mockSession)} }

func (s *mockStore) session(w http.ResponseWriter, r *http.Request) *mockSession {
	const cookieName = "weevcrafts_ui"
	cookie, err := r.Cookie(cookieName)
	id := ""
	if err == nil {
		id = cookie.Value
	}
	if id == "" {
		id = fmt.Sprintf("preview-%d", nextSessionID())
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions[id] == nil {
		s.sessions[id] = &mockSession{Wishlist: map[string]bool{"madhubani-tree": true, "brass-elephant": true, "chanderi-royal": true, "wooden-box": true, "ceramic-mugs": true, "blue-tote": true}, Cart: seedCart(), Orders: seedOrders()}
	}
	return s.sessions[id]
}

var sessionCounter int64
var sessionCounterMu sync.Mutex

func nextSessionID() int64 {
	sessionCounterMu.Lock()
	defer sessionCounterMu.Unlock()
	sessionCounter++
	return sessionCounter
}

func (s *mockStore) toggleWishlist(w http.ResponseWriter, r *http.Request, slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessionUnlocked(w, r)
	session.Wishlist[slug] = !session.Wishlist[slug]
}

func (s *mockStore) updateCart(w http.ResponseWriter, r *http.Request, slug string, delta int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessionUnlocked(w, r)
	for i := range session.Cart {
		if session.Cart[i].Product.Slug == slug {
			session.Cart[i].Quantity += delta
			if session.Cart[i].Quantity < 1 {
				session.Cart = append(session.Cart[:i], session.Cart[i+1:]...)
			}
			return
		}
	}
	if delta > 0 {
		if product, ok := productBySlug(slug); ok {
			session.Cart = append(session.Cart, viewmodels.CartItem{Product: product, Quantity: 1, Delivery: "4 - 6 days"})
		}
	}
}

func (s *mockStore) moveCartToWishlist(w http.ResponseWriter, r *http.Request, slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessionUnlocked(w, r)
	for i := range session.Cart {
		if session.Cart[i].Product.Slug == slug {
			session.Wishlist[slug] = true
			session.Cart = append(session.Cart[:i], session.Cart[i+1:]...)
			return
		}
	}
}

func (s *mockStore) sessionUnlocked(w http.ResponseWriter, r *http.Request) *mockSession {
	const cookieName = "weevcrafts_ui"
	cookie, err := r.Cookie(cookieName)
	id := ""
	if err == nil {
		id = cookie.Value
	}
	if id == "" {
		id = fmt.Sprintf("preview-%d", nextSessionID())
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	if s.sessions[id] == nil {
		s.sessions[id] = &mockSession{Wishlist: map[string]bool{"madhubani-tree": true, "brass-elephant": true, "chanderi-royal": true, "wooden-box": true, "ceramic-mugs": true, "blue-tote": true}, Cart: seedCart(), Orders: seedOrders()}
	}
	return s.sessions[id]
}

func cartCount(items []viewmodels.CartItem) int {
	count := 0
	for _, item := range items {
		count += item.Quantity
	}
	return count
}

func parseDelta(r *http.Request) int {
	if quantity, err := strconv.Atoi(r.FormValue("quantity")); err == nil && quantity > 0 {
		if quantity > 99 {
			return 99
		}
		return quantity
	}
	value, _ := strconv.Atoi(r.FormValue("delta"))
	if value == 0 {
		value = 1
	}
	if value < -1 {
		return -1
	}
	return value
}

func slugFromPath(path string) string {
	return strings.Trim(strings.TrimPrefix(path, "/ui/"), "/")
}
