package models

import (
	"time"

	"github.com/h44z/wg-portal/internal/domain"
)

// UserInterfacePool is the JSON view of one (user × interface) pool row
// (BNet-m76e). v1 mirror of v0/model.UserInterfacePool.
type UserInterfacePool struct {
	UserIdentifier      string `json:"UserIdentifier" example:"rashed"`
	InterfaceIdentifier string `json:"InterfaceIdentifier" example:"wg0"`

	PoolV4    string `json:"PoolV4" binding:"omitempty,cidr" example:"10.66.1.0/27"`
	PoolV6Ula string `json:"PoolV6Ula,omitempty" binding:"omitempty,cidr" example:"fdcc:ad94:bacf:6160:0:1::/80"`
	PoolV6Pi  string `json:"PoolV6Pi,omitempty" binding:"omitempty,cidr" example:"2602:f481:0:cc:1::/80"`

	UpdatedAt *time.Time `json:"UpdatedAt,omitempty" readonly:"true"`
	UpdatedBy string     `json:"UpdatedBy,omitempty" readonly:"true"`
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

// UserPoolUpdateRequest is the body of PUT /user/by-id/{id}/pools/{iface}.
type UserPoolUpdateRequest struct {
	PoolV4           string `json:"PoolV4" binding:"omitempty,cidr"`
	PoolV6Ula        string `json:"PoolV6Ula" binding:"omitempty,cidr"`
	PoolV6Pi         string `json:"PoolV6Pi" binding:"omitempty,cidr"`
	SkipPeerRenumber bool   `json:"SkipPeerRenumber,omitempty"`
}
