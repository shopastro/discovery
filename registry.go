package discovery

import (
	"github.com/yousinn/registry"
	"google.golang.org/grpc/resolver"
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
		exit  chan struct{}
	}
)

func NewRegistry(reg registry.Registry, ttl int) *Registry {
	return &Registry{
		ttl:   ttl,
		reg:   reg,
		exit:  make(chan struct{}),
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

	<-r.exit
}

func (r *Registry) DiscoveryEnabled() {
	resolver.Register(NewDiscovery(r.reg))
}

func (r *Registry) keepAlive(svc *registry.Service) {
	t := time.NewTicker(time.Duration(r.ttl) * time.Millisecond)

	for {
		select {
		case <-t.C:
			if err := r.reg.Register(svc, registry.RegisterTTL(time.Duration(r.ttl))); err != nil {
				log.Println(err)
			}
		case <-r.close:
			if err := r.reg.Deregister(svc); err != nil {
				log.Println(err)
			}

			r.exit <- struct{}{}
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
