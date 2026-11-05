// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"encoding/binary"
	"strings"
)

// ID uniquely identifies a widget across frames. Zero indicates no widget.
type ID uint64

const (
	fnvOffset64 uint64 = 14695981039346656037
	fnvPrime64  uint64 = 1099511628211
)

func hashString(seed ID, s string) ID {
	h := uint64(seed)
	if h == 0 {
		h = fnvOffset64
	}
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	if h == 0 {
		h = 1
	}
	return ID(h)
}

func hashInt(seed ID, v int) ID {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	h := uint64(seed)
	if h == 0 {
		h = fnvOffset64
	}
	for _, b := range buf {
		h ^= uint64(b)
		h *= fnvPrime64
	}
	if h == 0 {
		h = 1
	}
	return ID(h)
}

// ParseLabel follows Dear ImGui conventions for separating visible text from
// the hash key:
//   - "Label" displays "Label" and hashes "Label".
//   - "Label##Hidden" displays "Label" and hashes "Label##Hidden".
//   - "Dynamic Text###StableID" displays "Dynamic Text" and hashes "###StableID".
func ParseLabel(label string) (display, hashKey string) {
	if idx := strings.Index(label, "###"); idx >= 0 {
		return label[:idx], label[idx:]
	}
	if idx := strings.Index(label, "##"); idx >= 0 {
		return label[:idx], label
	}
	return label, label
}

// Storage retains lightweight internal widget state across frames keyed by widget ID.
type Storage struct {
	bools  map[ID]bool
	ints   map[ID]int
	floats map[ID]float64
	points map[ID]Point
}

func newStorage() *Storage {
	return &Storage{
		bools:  make(map[ID]bool),
		ints:   make(map[ID]int),
		floats: make(map[ID]float64),
		points: make(map[ID]Point),
	}
}

// Bool returns the stored boolean for id, initializing it to defaultVal if absent.
func (s *Storage) Bool(id ID, defaultVal bool) bool {
	if v, ok := s.bools[id]; ok {
		return v
	}
	s.bools[id] = defaultVal
	return defaultVal
}

// SetBool stores a boolean value for id.
func (s *Storage) SetBool(id ID, val bool) {
	s.bools[id] = val
}

// Int returns the stored integer for id, initializing it to defaultVal if absent.
func (s *Storage) Int(id ID, defaultVal int) int {
	if v, ok := s.ints[id]; ok {
		return v
	}
	s.ints[id] = defaultVal
	return defaultVal
}

// SetInt stores an integer value for id.
func (s *Storage) SetInt(id ID, val int) {
	s.ints[id] = val
}

// Float returns the stored float64 for id, initializing it to defaultVal if absent.
func (s *Storage) Float(id ID, defaultVal float64) float64 {
	if v, ok := s.floats[id]; ok {
		return v
	}
	s.floats[id] = defaultVal
	return defaultVal
}

// SetFloat stores a float64 value for id.
func (s *Storage) SetFloat(id ID, val float64) {
	s.floats[id] = val
}

// Point returns the stored Point for id, initializing it to defaultVal if absent.
func (s *Storage) Point(id ID, defaultVal Point) Point {
	if v, ok := s.points[id]; ok {
		return v
	}
	s.points[id] = defaultVal
	return defaultVal
}

// SetPoint stores a Point value for id.
func (s *Storage) SetPoint(id ID, val Point) {
	s.points[id] = val
}
