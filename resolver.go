package discovery

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/shopastro/registry"
	"google.golang.org/grpc/resolver"
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
		log.Println(fmt.Sprintf("init sync name: %s", r.name), err)
	}

	defer r.wg.Done()

	w, err := r.reg.Watch(registry.WatchContext(r.ctx))
	if err != nil {
		log.Println(fmt.Sprintf("ticker sync name: %s", r.name), err)
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
				log.Println(fmt.Sprintf("ticker sync name: %s", r.name), err)
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

	if err := r.cc.UpdateState(resolver.State{
		Addresses: addrs,
	}); err != nil {
		log.Println(fmt.Sprintf("[resolver delete. name: %s]", r.name), err)
	}
}

func (r *Resolver) update(svc *registry.Service) {
	for _, node := range svc.Nodes {
		addr := resolver.Address{
			Addr:       node.Address,
			ServerName: svc.Name,
		}

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

	return r.cc.UpdateState(resolver.State{Addresses: addr})
}

func (r *Resolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *Resolver) Close() {
	r.cancel()

	r.wg.Wait()
}
