package lru

import (
    "container/list"
    "crypto/md5"
    "fmt"
    "log"
    "sync"
)

// node type holds the actual value and it's key - this allows removal of the LRU element
// from the linked list and then to delete it from the map with the key.
type node struct {
    key   string
    value []byte
    md5   string
}

// LRU is the basic implementation of an LRU cache
type LRU struct {
    rw      *sync.RWMutex
    dict    map[string]*list.Element
    list    *list.List
    size    int64
    maxsize int64
}

// Set will add/replace the given key with the specified value and return the calculated md5 hash
func (l *LRU) Set(key string, value []byte) string {
    l.rw.Lock()
    defer l.rw.Unlock()

    n := &node{
        key:   key,
        value: value,
        md5:   fmt.Sprintf("%x", md5.Sum(value)),
    }

    elm := l.list.PushFront(n)
    l.dict[key] = elm
    l.size += int64(len(value))

    if l.size > l.maxsize {
        for l.size > l.maxsize {
            elm := l.list.Back()
            n := elm.Value.(*node)
            delete(l.dict, n.key)

            l.size -= int64(len(n.value))
            l.list.Remove(elm)
        }
    }

    return n.md5
}

// Get will fetch the value for the given key or return nil if it does not exist
func (l *LRU) Get(key string) ([]byte, string) {
    l.rw.Lock()
    defer l.rw.Unlock()

    if e, ok := l.dict[key]; ok {
        n := e.Value.(*node)
        l.list.MoveToFront(e)
        return n.value, n.md5
    }

    return nil, ""
}

// Exists will determine if there is an entry for the given key
func (l *LRU) Exists(key string) bool {
    l.rw.RLock()
    defer l.rw.RUnlock()

    _, ok := l.dict[key]

    return ok
}

// Delete will remove an entry for the given key if it exists
func (l *LRU) Delete(key string) {
    l.rw.Lock()
    defer l.rw.Unlock()

    elm, ok := l.dict[key]

    if !ok {
        return
    }

    delete(l.dict, key)
    l.list.Remove(elm)
}

// Status reports the current state of the cache returning:
// + number of elements
// + current byte size
// + maximum byte size
func (l *LRU) Status() (int, int64, int64) {
    l.rw.RLock()
    defer l.rw.RUnlock()
    return len(l.dict), l.size, l.maxsize
}

// New will create a LRU instance with the given size of elements
func New(maxsize int64) *LRU {
    log.Printf("created a new cache: max maxsize = %d bytes", maxsize)
    lru := &LRU{
        rw:      &sync.RWMutex{},
        dict:    map[string]*list.Element{},
        list:    list.New(),
        size:    0,
        maxsize: maxsize,
    }

    return lru
}
