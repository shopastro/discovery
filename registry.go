package discovery

import (
	"github.com/shopastro/registry"
	"log"
	"time"
)

type (
	Registry struct {
		ttl   int
		reg   registry.Registry
		close chan struct{}
	}
)

func NewRegistry(reg registry.Registry, ttl int) *Registry {
	return &Registry{
		ttl:   ttl,
		reg:   reg,
		close: make(chan struct{}),
	}
}

func (r *Registry) Register(svc *registry.Service) error {
	if err := r.reg.Register(svc, registry.RegisterTTL(time.Duration(r.ttl))); err != nil {
		return err
	}

	go r.keepAlive(svc)
	return nil
}

func (r *Registry) Stop() {
	r.close <- struct{}{}
}

func (r *Registry) keepAlive(svc *registry.Service) {
	t := time.NewTicker(time.Duration(r.ttl) * time.Second)

	for {
		select {
		case <-t.C:
			if err := r.reg.Register(svc, registry.RegisterTTL(time.Duration(r.ttl))); err != nil {
				log.Println(err)
			}
		case <-r.close:
			return
		}
	}
}
