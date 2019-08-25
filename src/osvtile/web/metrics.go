package web

import (
    "net/http"
    "sync"
    "time"
)

type Metrics struct {
    rw       *sync.RWMutex
    Requests int64            `json:"requests"`
    Status   map[int]int64    `json:"status"`
    Methods  map[string]int64 `json:"methods"`
    Start    time.Time        `json:"start"`
}

func (m *Metrics) Log(r *http.Request, status int) {
    m.rw.Lock()
    defer m.rw.Unlock()
    m.Requests++

    if _, ok := m.Status[status]; ok {
        m.Status[status]++
    } else {
        m.Status[status] = 1
    }

    if _, ok := m.Methods[r.Method]; ok {
        m.Methods[r.Method]++
    } else {
        m.Methods[r.Method] = 1
    }
}

func NewMetrics() *Metrics {
    return &Metrics{
        rw:       &sync.RWMutex{},
        Requests: 0,
        Status:   map[int]int64{},
        Methods:  map[string]int64{},
        Start:    time.Now().UTC(),
    }
}
