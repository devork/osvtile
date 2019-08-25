package lru

import (
    "crypto/md5"
    "fmt"
    "github.com/stretchr/testify/require"
    "testing"
)

func TestLRU_GetSet(t *testing.T) {
    cache := New(1024)

    cache.Set("a", []byte("aaaaaaaa"))
    cache.Set("b", []byte("bbbbbbbb"))
    cache.Set("c", []byte("cccccccc"))

    v, m := cache.Get("a")
    require.Equal(t, []byte("aaaaaaaa"), v)
    require.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte("aaaaaaaa"))), m)

    v, m = cache.Get("b")
    require.Equal(t, []byte("bbbbbbbb"), v)
    require.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte("bbbbbbbb"))), m)

    v, m = cache.Get("c")
    require.Equal(t, []byte("cccccccc"), v)
    require.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte("cccccccc"))), m)
}

func TestLRU_Replace(t *testing.T) {
    cache := New(1024)

    cache.Set("a", []byte("aaaaaaaa"))

    v, _ := cache.Get("a")
    require.Equal(t, []byte("aaaaaaaa"), v)

    cache.Set("a", []byte("bbbbbbbb"))
    v, _ = cache.Get("a")
    require.Equal(t, []byte("bbbbbbbb"), v)
}

func TestLRU_GetNNonExistent(t *testing.T) {
    cache := New(1024)

    cache.Set("a", []byte("aaaaaaaa"))

    v, m := cache.Get("b")
    require.Nil(t, v)
    require.Empty(t, m)
    require.False(t, cache.Exists("b"))

    require.True(t, cache.Exists("a"))
}

func TestLRU_Eviction(t *testing.T) {
    cache := New(1024)

    data0 := make([]byte, 256, 256)

    cache.Set("a", data0)
    cache.Set("b", data0)
    cache.Set("c", data0)
    cache.Set("d", data0)

    _, _ = cache.Get("d")
    _, _ = cache.Get("c")
    _, _ = cache.Get("b")
    _, _ = cache.Get("a")

    require.Equal(t, "a", cache.list.Front().Value.(*node).key)
    require.Equal(t, "d", cache.list.Back().Value.(*node).key)
    require.Equal(t, int64(1024), cache.size)

    cache.Set("e", data0)
    require.False(t, cache.Exists("d"))
    require.True(t, cache.Exists("e"))

    require.Equal(t, "e", cache.list.Front().Value.(*node).key)
    require.Equal(t, "c", cache.list.Back().Value.(*node).key)
}

func TestLRU_Clear(t *testing.T) {
    cache := New(1024)

    data0 := make([]byte, 256, 256)

    cache.Set("a", data0)
    cache.Set("b", data0)
    cache.Set("c", data0)
    cache.Set("d", data0)

    require.Equal(t, 4, cache.list.Len())
    require.Equal(t, 4, len(cache.dict))
    require.Equal(t, int64(1024), cache.size)
    require.True(t, cache.Exists("a"))
    require.True(t, cache.Exists("b"))
    require.True(t, cache.Exists("c"))
    require.True(t, cache.Exists("d"))

    cache.Clear()
    require.Equal(t, 0, cache.list.Len())
    require.Equal(t, 0, len(cache.dict))
    require.Equal(t, int64(0), cache.size)
    require.False(t, cache.Exists("a"))
    require.False(t, cache.Exists("b"))
    require.False(t, cache.Exists("c"))
    require.False(t, cache.Exists("d"))
}

func TestLRU_Status(t *testing.T) {
    cache := New(1024)

    data0 := make([]byte, 256, 256)

    cache.Set("a", data0)
    cache.Set("b", data0)
    cache.Set("c", data0)

    status := cache.Status()
    require.Equal(t, 3, status.Elements)
    require.Equal(t, int64(768), status.Size)
    require.Equal(t, int64(1024), status.MaxSize)

    cache.Clear()

    status = cache.Status()
    require.Equal(t, 0, status.Elements)
    require.Equal(t, int64(0), status.Size)
    require.Equal(t, int64(1024), status.MaxSize)

}
