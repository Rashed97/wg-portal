package model

import (
	"time"

	"github.com/h44z/wg-portal/internal/domain"
)

// UserInterfacePool is the JSON view of one (user × interface) pool row.
// Returned by GET /user/{id}/pools as a list, and individually by
// PUT /user/{id}/pools/{iface}. (BNet-m76e.)
type UserInterfacePool struct {
	UserIdentifier      string `json:"UserIdentifier"`
	InterfaceIdentifier string `json:"InterfaceIdentifier"`

	PoolV4    string `json:"PoolV4"`              // e.g. "10.66.1.0/27"
	PoolV6Ula string `json:"PoolV6Ula,omitempty"` // e.g. "fdcc:ad94:bacf:6160:0:1::/80"
	PoolV6Pi  string `json:"PoolV6Pi,omitempty"`  // e.g. "2602:f481:0:cc:1::/80"

	UpdatedAt *time.Time `json:"UpdatedAt,omitempty"`
	UpdatedBy string     `json:"UpdatedBy,omitempty"`
}

func NewUserInterfacePool(p *domain.UserInterfacePool) *UserInterfacePool {
	return &UserInterfacePool{
		UserIdentifier:      string(p.UserIdentifier),
		InterfaceIdentifier: string(p.InterfaceIdentifier),
		PoolV4:              p.PoolV4,
		PoolV6Ula:           p.PoolV6Ula,
		PoolV6Pi:            p.PoolV6Pi,
		UpdatedAt:           &p.UpdatedAt,
		UpdatedBy:           p.UpdatedBy,
	}
}

func NewUserInterfacePools(src []domain.UserInterfacePool) []UserInterfacePool {
	out := make([]UserInterfacePool, len(src))
	for i := range src {
		out[i] = *NewUserInterfacePool(&src[i])
	}
	return out
}

// UserPoolUpdateRequest is the body of PUT /user/{id}/pools/{iface}.
// Empty CIDR strings clear that family. SkipPeerRenumber=true means
// "change the pool row but DON'T rewrite existing peer addresses"
// (escape hatch for special migration scenarios).
type UserPoolUpdateRequest struct {
	PoolV4           string `json:"PoolV4"`
	PoolV6Ula        string `json:"PoolV6Ula"`
	PoolV6Pi         string `json:"PoolV6Pi"`
	SkipPeerRenumber bool   `json:"SkipPeerRenumber,omitempty"`
}

// PoolAllocatorState is the read-only summary of an interface's
// auto-allocator state — for the admin UI's "Allocator state" panel.
type PoolAllocatorState struct {
	InterfaceIdentifier string `json:"InterfaceIdentifier"`
	AllocatedCount      int    `json:"AllocatedCount"`

	NextFreeV4    string `json:"NextFreeV4,omitempty"`
	NextFreeV6Ula string `json:"NextFreeV6Ula,omitempty"`
	NextFreeV6Pi  string `json:"NextFreeV6Pi,omitempty"`

	SupernetV4Total       int `json:"SupernetV4Total"`
	SupernetV6UlaTotal    int `json:"SupernetV6UlaTotal"`
	SupernetV6PiTotal     int `json:"SupernetV6PiTotal"`
	SupernetV4Reserved    int `json:"SupernetV4Reserved"`
	SupernetV6UlaReserved int `json:"SupernetV6UlaReserved"`
	SupernetV6PiReserved  int `json:"SupernetV6PiReserved"`
}
