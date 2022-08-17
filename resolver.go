package discovery

import (
	"context"
	"fmt"
	"github.com/yousinn/registry"
	"google.golang.org/grpc/resolver"
	"log"
	"sync"
	"time"
)

type (
	Resolver struct {
		name           string
		cc             resolver.ClientConn
		ctx            context.Context
		reg            registry.Registry
		addrsCacheList []resolver.Address
		wg             sync.WaitGroup
		cancel         context.CancelFunc
	}
)

func (r *Resolver) watch() {
	defer r.wg.Done()

	w, err := r.reg.Watch(registry.WatchContext(r.ctx))
	if err != nil {
		return
	}

	ups := make(chan []*registry.Result)
	go w.Next(ups)

	t := time.NewTicker(time.Second)

	for {
		select {
		case <-r.ctx.Done():
			return

		case resp := <-ups:
			r.resolver(resp)
			fmt.Println("watch start ...")

		case <-t.C:
			if err := r.sync(); err != nil {
				log.Println("sync", err)
			}

			fmt.Println("watch ticker start ...")
		}
	}
}

func (r *Resolver) resolver(resp []*registry.Result) {
	for _, res := range resp {
		switch res.Action {
		case "create", "update":
			r.update(res.Service)
		case "delete":
			r.delete(res.Service)
		}
	}
}

func (r *Resolver) update(svc *registry.Service) {
	for _, node := range svc.Nodes {
		addr := resolver.Address{
			Addr:       node.Address,
			ServerName: svc.Name,
		}

		if !r.exist(r.addrsCacheList, addr) {
			r.addrsCacheList = append(r.addrsCacheList, addr)

			if err := r.cc.UpdateState(resolver.State{Addresses: r.addrsCacheList}); err != nil {
				log.Println(err)
			}
		}
	}
}

func (r *Resolver) delete(svc *registry.Service) {
	for _, node := range svc.Nodes {
		addr := resolver.Address{
			Addr:       node.Address,
			ServerName: svc.Name,
		}

		if addrs, ok := r.remove(r.addrsCacheList, addr); ok {
			r.addrsCacheList = addrs

			if err := r.cc.UpdateState(resolver.State{Addresses: r.addrsCacheList}); err != nil {
				log.Println(err)
			}
		}
	}
}

func (r *Resolver) sync() error {
	resp, err := r.reg.GetService(r.name, registry.GetContext(r.ctx))
	if err != nil {
		return err
	}

	for _, v := range resp {
		for _, node := range v.Nodes {
			r.addrsCacheList = append(r.addrsCacheList, resolver.Address{Addr: node.Address})
		}
	}

	return r.cc.UpdateState(resolver.State{Addresses: r.addrsCacheList})
}

func (r *Resolver) exist(address []resolver.Address, addr resolver.Address) bool {
	for v := range address {
		if address[v].Addr == addr.Addr {
			return true
		}
	}

	return false
}

func (r *Resolver) remove(s []resolver.Address, addr resolver.Address) ([]resolver.Address, bool) {
	for i := range s {
		if s[i].Addr == addr.Addr {
			s[i] = s[len(s)-1]
			return s[:len(s)-1], true
		}
	}

	return nil, false
}

func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {
	r.cancel()

	r.wg.Wait()
}
