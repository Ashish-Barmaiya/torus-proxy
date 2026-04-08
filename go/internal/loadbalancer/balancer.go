package loadbalancer

import (
	"torus-proxy/internal/upstream"
)

type LoadBalancer interface {
	Next() *upstream.Backend	
}