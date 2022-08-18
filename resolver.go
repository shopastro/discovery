package discovery

import (
	"context"
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
		addrsCacheList sync.Map
		wg             sync.WaitGroup
		cancel         context.CancelFunc
	}
)

func (r *Resolver) watch() {
	if err := r.sync(); err != nil {
		log.Println("init sync", err)
	}

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

		case <-t.C:
			if err := r.sync(); err != nil {
				log.Println("sync", err)
			}
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

func (r *Resolver) delete(svc *registry.Service) {
	r.addrsCacheList.Delete(svc.Key)

	var addrs []resolver.Address
	r.addrsCacheList.Range(func(key, value any) bool {
		if addr, ok := value.(resolver.Address); ok {
			addrs = append(addrs, addr)
		}

		return true
	})

	log.Println("[resolver delete]", addrs)
	if err := r.cc.UpdateState(resolver.State{
		Addresses: addrs,
	}); err != nil {
		log.Println("[resolver delete]", err)
	}
}

func (r *Resolver) update(svc *registry.Service) {
	for _, node := range svc.Nodes {
		addr := resolver.Address{
			Addr:       node.Address,
			ServerName: svc.Name,
		}

		log.Println("[resolver update]", addr)
		r.addrsCacheList.LoadOrStore(svc.Key, addr)
	}
}

func (r *Resolver) sync() error {
	resp, err := r.reg.GetService(r.name, registry.GetContext(r.ctx))
	if err != nil {
		return err
	}

	var addr []resolver.Address

	for _, v := range resp {
		for _, node := range v.Nodes {
			rsv := resolver.Address{Addr: node.Address}

			addr = append(addr, rsv)

			r.addrsCacheList.Store(v.Key, rsv)
		}
	}

	log.Println("[resolver sync]", addr)
	return r.cc.UpdateState(resolver.State{Addresses: addr})
}

func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {
	r.cancel()

	r.wg.Wait()
}
