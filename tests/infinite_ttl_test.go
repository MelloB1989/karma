package tests

// import (
// 	"testing"
// 	"time"
// )

// func TestInfiniteTTLConstant(t *testing.T) {
// 	if InfiniteTTL >= 0 {
// 		t.Errorf("InfiniteTTL should be negative, got %v", InfiniteTTL)
// 	}
// }

// func TestIsInfiniteTTL(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		ttl      time.Duration
// 		expected bool
// 	}{
// 		{
// 			name:     "InfiniteTTL constant",
// 			ttl:      InfiniteTTL,
// 			expected: true,
// 		},
// 		{
// 			name:     "Negative duration",
// 			ttl:      -1 * time.Second,
// 			expected: true,
// 		},
// 		{
// 			name:     "Zero duration",
// 			ttl:      0,
// 			expected: false,
// 		},
// 		{
// 			name:     "Positive duration",
// 			ttl:      5 * time.Minute,
// 			expected: false,
// 		},
// 		{
// 			name:     "Very large duration",
// 			ttl:      100 * time.Hour,
// 			expected: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			result := IsInfiniteTTL(tt.ttl)
// 			if result != tt.expected {
// 				t.Errorf("IsInfiniteTTL(%v) = %v, expected %v", tt.ttl, result, tt.expected)
// 			}
// 		})
// 	}
// }

// func TestMemoryCacheInfiniteTTL(t *testing.T) {
// 	cache := newMemoryCache()
// 	key := "test_infinite_key"
// 	data := []byte("test data")

// 	// Set with infinite TTL
// 	cache.Set(key, data, InfiniteTTL)

// 	// Immediately check if data exists
// 	retrievedData, exists := cache.Get(key)
// 	if !exists {
// 		t.Error("Data should exist immediately after setting with infinite TTL")
// 	}
// 	if string(retrievedData) != string(data) {
// 		t.Errorf("Retrieved data mismatch: expected %s, got %s", string(data), string(retrievedData))
// 	}

// 	// Sleep a bit and check again (simulating time passing)
// 	time.Sleep(10 * time.Millisecond)
// 	retrievedData, exists = cache.Get(key)
// 	if !exists {
// 		t.Error("Data with infinite TTL should still exist after time has passed")
// 	}
// 	if string(retrievedData) != string(data) {
// 		t.Errorf("Retrieved data mismatch after time: expected %s, got %s", string(data), string(retrievedData))
// 	}

// 	// Test manual deletion
// 	cache.Delete(key)
// 	_, exists = cache.Get(key)
// 	if exists {
// 		t.Error("Data should not exist after manual deletion")
// 	}
// }

// func TestMemoryCacheRegularTTL(t *testing.T) {
// 	cache := newMemoryCache()
// 	key := "test_regular_key"
// 	data := []byte("test data")

// 	// Set with very short TTL
// 	cache.Set(key, data, 5*time.Millisecond)

// 	// Immediately check if data exists
// 	retrievedData, exists := cache.Get(key)
// 	if !exists {
// 		t.Error("Data should exist immediately after setting")
// 	}
// 	if string(retrievedData) != string(data) {
// 		t.Errorf("Retrieved data mismatch: expected %s, got %s", string(data), string(retrievedData))
// 	}

// 	// Wait for expiration
// 	time.Sleep(10 * time.Millisecond)
// 	_, exists = cache.Get(key)
// 	if exists {
// 		t.Error("Data should have expired after TTL")
// 	}
// }

// func TestORMHasInfiniteTTL(t *testing.T) {
// 	type TestStruct struct {
// 		TableName string `karma_table:"test"`
// 		ID        int    `json:"id"`
// 	}

// 	tests := []struct {
// 		name     string
// 		options  []Options
// 		expected bool
// 	}{
// 		{
// 			name:     "No TTL configured",
// 			options:  []Options{},
// 			expected: false,
// 		},
// 		{
// 			name: "Regular TTL configured",
// 			options: []Options{
// 				WithCacheTTL(5 * time.Minute),
// 			},
// 			expected: false,
// 		},
// 		{
// 			name: "Infinite TTL with constant",
// 			options: []Options{
// 				WithCacheTTL(InfiniteTTL),
// 			},
// 			expected: true,
// 		},
// 		{
// 			name: "Infinite TTL with convenience function",
// 			options: []Options{
// 				WithInfiniteCacheTTL(),
// 			},
// 			expected: true,
// 		},
// 		{
// 			name: "Negative TTL",
// 			options: []Options{
// 				WithCacheTTL(-1 * time.Second),
// 			},
// 			expected: true,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			orm := Load(&TestStruct{}, tt.options...)
// 			if orm == nil {
// 				t.Fatal("Failed to create ORM")
// 			}

// 			result := orm.HasInfiniteTTL()
// 			if result != tt.expected {
// 				t.Errorf("HasInfiniteTTL() = %v, expected %v", result, tt.expected)
// 			}
// 		})
// 	}
// }

// func TestWithInfiniteCacheTTLOption(t *testing.T) {
// 	type TestStruct struct {
// 		TableName string `karma_table:"test"`
// 		ID        int    `json:"id"`
// 	}

// 	orm := Load(&TestStruct{}, WithInfiniteCacheTTL())
// 	if orm == nil {
// 		t.Fatal("Failed to create ORM")
// 	}

// 	if orm.CacheTTL == nil {
// 		t.Error("CacheTTL should be set when using WithInfiniteCacheTTL()")
// 		return
// 	}

// 	if *orm.CacheTTL != InfiniteTTL {
// 		t.Errorf("CacheTTL should be InfiniteTTL, got %v", *orm.CacheTTL)
// 	}

// 	if !orm.HasInfiniteTTL() {
// 		t.Error("HasInfiniteTTL() should return true when using WithInfiniteCacheTTL()")
// 	}
// }

// func TestWithCacheTTLOption(t *testing.T) {
// 	type TestStruct struct {
// 		TableName string `karma_table:"test"`
// 		ID        int    `json:"id"`
// 	}

// 	testTTL := InfiniteTTL
// 	orm := Load(&TestStruct{}, WithCacheTTL(testTTL))
// 	if orm == nil {
// 		t.Fatal("Failed to create ORM")
// 	}

// 	if orm.CacheTTL == nil {
// 		t.Error("CacheTTL should be set when using WithCacheTTL()")
// 		return
// 	}

// 	if *orm.CacheTTL != testTTL {
// 		t.Errorf("CacheTTL should be %v, got %v", testTTL, *orm.CacheTTL)
// 	}

// 	if !orm.HasInfiniteTTL() {
// 		t.Error("HasInfiniteTTL() should return true when TTL is InfiniteTTL")
// 	}
// }

// func TestMemoryCacheZeroTime(t *testing.T) {
// 	cache := newMemoryCache()
// 	key := "test_zero_time"
// 	data := []byte("test data")

// 	// Manually set with zero time (infinite TTL)
// 	cache.mutex.Lock()
// 	cache.data[key] = data
// 	cache.ttl[key] = time.Time{} // Zero time indicates infinite TTL
// 	cache.mutex.Unlock()

// 	// Check that data exists and doesn't expire
// 	retrievedData, exists := cache.Get(key)
// 	if !exists {
// 		t.Error("Data with zero time (infinite TTL) should exist")
// 	}
// 	if string(retrievedData) != string(data) {
// 		t.Errorf("Retrieved data mismatch: expected %s, got %s", string(data), string(retrievedData))
// 	}

// 	// Wait and check again
// 	time.Sleep(10 * time.Millisecond)
// 	retrievedData, exists = cache.Get(key)
// 	if !exists {
// 		t.Error("Data with zero time (infinite TTL) should still exist after time")
// 	}
// }

// func BenchmarkMemoryCacheInfiniteTTL(b *testing.B) {
// 	cache := newMemoryCache()
// 	data := []byte("benchmark data")

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		key := "bench_key_" + string(rune(i))
// 		cache.Set(key, data, InfiniteTTL)
// 		cache.Get(key)
// 	}
// }

// func BenchmarkMemoryCacheRegularTTL(b *testing.B) {
// 	cache := newMemoryCache()
// 	data := []byte("benchmark data")

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		key := "bench_key_" + string(rune(i))
// 		cache.Set(key, data, 5*time.Minute)
// 		cache.Get(key)
// 	}
// }
