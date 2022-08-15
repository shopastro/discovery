package discovery

import (
	"github.com/shopastro/registry"
	"log"
	"net"
	"strings"
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

func (r *Registry) Address(addr string) (string, error) {
	var (
		err              error
		advt, host, port string
	)

	if len(addr) > 0 {
		advt = addr
	}

	if cnt := strings.Count(advt, ":"); cnt >= 1 {
		host, port, err = net.SplitHostPort(advt)
		if err != nil {
			return "", err
		}
	} else {
		host = advt
	}

	if ip := net.ParseIP(host); ip != nil {

	}

	newAddr, err := Extract(host)
	if err != nil {
		return "", err
	}

	if port != "" {
		newAddr = HostPort(newAddr, port)
	}

	return newAddr, nil
}
