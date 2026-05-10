package models

import "github.com/h44z/wg-portal/internal/domain"

// PoolAllocatorState is the read-only allocator-state DTO returned by
// GET /interface/pool-state/{id} (BNet-m76e QoL).
type PoolAllocatorState struct {
	InterfaceIdentifier string `json:"InterfaceIdentifier"`
	AllocatedCount      int    `json:"AllocatedCount"`

	NextFreeV4    string `json:"NextFreeV4"`
	NextFreeV6Ula string `json:"NextFreeV6Ula"`
	NextFreeV6Pi  string `json:"NextFreeV6Pi"`

	SupernetV4Total       int `json:"SupernetV4Total"`
	SupernetV6UlaTotal    int `json:"SupernetV6UlaTotal"`
	SupernetV6PiTotal     int `json:"SupernetV6PiTotal"`
	SupernetV4Reserved    int `json:"SupernetV4Reserved"`
	SupernetV6UlaReserved int `json:"SupernetV6UlaReserved"`
	SupernetV6PiReserved  int `json:"SupernetV6PiReserved"`
}

func NewPoolAllocatorState(s *domain.PoolAllocatorState) *PoolAllocatorState {
	if s == nil {
		return nil
	}
	return &PoolAllocatorState{
		InterfaceIdentifier:   string(s.InterfaceIdentifier),
		AllocatedCount:        s.AllocatedCount,
		NextFreeV4:            s.NextFreeV4,
		NextFreeV6Ula:         s.NextFreeV6Ula,
		NextFreeV6Pi:          s.NextFreeV6Pi,
		SupernetV4Total:       s.SupernetV4Total,
		SupernetV6UlaTotal:    s.SupernetV6UlaTotal,
		SupernetV6PiTotal:     s.SupernetV6PiTotal,
		SupernetV4Reserved:    s.SupernetV4Reserved,
		SupernetV6UlaReserved: s.SupernetV6UlaReserved,
		SupernetV6PiReserved:  s.SupernetV6PiReserved,
	}
}
