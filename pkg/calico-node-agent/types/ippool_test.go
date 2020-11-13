package types

import "testing"

func TestNodeIPPool_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cidr    string
		wantErr bool
	}{
		{
			name:    "192.168.1.0/24 is validate",
			cidr:    "192.168.1.0/24",
			wantErr: false,
		},
		{
			name:    "192.168.1/24 is invalidate",
			cidr:    "192.168.1/24",
			wantErr: true,
		},
		{
			name:    "192.168.1.1 is validate",
			cidr:    "192.168.1.1",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NodeIPPool{
				CIDR: tt.cidr,
			}
			if err := pool.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("NodeIPPool.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
