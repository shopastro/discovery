package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yousinn/registry"
	"google.golang.org/grpc/resolver"
	"strings"
)

type (
	Discovery struct {
		reg registry.Registry
	}
)

func NewDiscovery(reg registry.Registry) resolver.Builder {
	return &Discovery{
		reg: reg,
	}
}

func (d *Discovery) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	body, err := json.Marshal(target.URL)

	fmt.Println("target.URL", body, err)
	r := &Resolver{
		cc:   cc,
		reg:  d.reg,
		name: d.name(target.URL.Path),
	}

	r.ctx, r.cancel = context.WithCancel(context.Background())

	r.wg.Add(1)
	go r.watch()

	return r, nil
}

func (d *Discovery) Scheme() string {
	return "etcd"
}

func (d *Discovery) name(target string) string {
	targets := strings.SplitN(target, "/", 2)
	if len(targets) >= 1 {
		switch len(targets) {
		case 1:
			if targets[0] != "" {
				hosts := strings.SplitN(targets[0], ":", 2)
				return hosts[0]
			}
		case 2:
			if targets[1] != "" {
				hosts := strings.SplitN(targets[1], ":", 2)
				return hosts[0]
			}
		}
	}

	return target
}
