package thumbs

import "sync"

type cachedThumb struct {
	Data  []byte
	Type  string
	ETag  string
	bytes int
}

type Mem struct {
	mu      sync.Mutex
	items   map[string]cachedThumb
	maxByte int
	used    int
}

func NewMem(maxBytes int) *Mem {
	if maxBytes <= 0 {
		maxBytes = 32 << 20
	}
	return &Mem{items: make(map[string]cachedThumb), maxByte: maxBytes}
}

func (m *Mem) Get(key string) (cachedThumb, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	item, ok := m.items[key]
	return item, ok
}

func (m *Mem) Put(key string, data []byte, contentType, etag string) {
	if len(data) == 0 || len(data) > m.maxByte {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.items[key]; ok {
		m.used -= old.bytes
		delete(m.items, key)
	}
	for m.used+len(data) > m.maxByte && len(m.items) > 0 {
		for k, v := range m.items {
			m.used -= v.bytes
			delete(m.items, k)
			break
		}
	}
	cp := append([]byte(nil), data...)
	m.items[key] = cachedThumb{Data: cp, Type: contentType, ETag: etag, bytes: len(cp)}
	m.used += len(cp)
}

func (m *Mem) Flush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.items = make(map[string]cachedThumb)
	m.used = 0
}

func (m *Mem) Delete(keys ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, key := range keys {
		if old, ok := m.items[key]; ok {
			m.used -= old.bytes
			delete(m.items, key)
		}
	}
}
