package fetcher

import (
	"log/slog"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

// SessionManager manages cookie/session state across requests per domain.
type SessionManager struct {
	jars   map[string]*cookiejar.Jar
	mu     sync.RWMutex
	logger *slog.Logger
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(logger *slog.Logger) *SessionManager {
	return &SessionManager{
		jars:   make(map[string]*cookiejar.Jar),
		logger: logger.With("component", "session_manager"),
	}
}

// GetJar returns the cookie jar for a domain, creating one if needed.
func (sm *SessionManager) GetJar(domain string) *cookiejar.Jar {
	sm.mu.RLock()
	jar, ok := sm.jars[domain]
	sm.mu.RUnlock()
	if ok {
		return jar
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Double-check
	jar, ok = sm.jars[domain]
	if ok {
		return jar
	}

	jar, _ = cookiejar.New(nil)
	sm.jars[domain] = jar
	return jar
}

// ClearDomain removes all cookies for a domain.
func (sm *SessionManager) ClearDomain(domain string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.jars, domain)
}

// ClearAll removes all cookies.
func (sm *SessionManager) ClearAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.jars = make(map[string]*cookiejar.Jar)
}

// DomainCount returns the number of domains with active sessions.
func (sm *SessionManager) DomainCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.jars)
}

// HasCookies checks if a domain has any cookies set.
func (sm *SessionManager) HasCookies(domain string) bool {
	sm.mu.RLock()
	jar, ok := sm.jars[domain]
	sm.mu.RUnlock()
	if !ok {
		return false
	}

	u, _ := url.Parse("https://" + domain)
	return len(jar.Cookies(u)) > 0
}
