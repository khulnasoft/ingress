package proxy

import (
	"net"

	"github.com/khulnasoft/kengine/v2"
	"github.com/pires/go-proxyproto"
)

var (
	_ = kengine.Provisioner(&Wrapper{})
	_ = kengine.Module(&Wrapper{})
	_ = kengine.ListenerWrapper(&Wrapper{})
)

func init() {
	kengine.RegisterModule(Wrapper{})
}

// Wrapper provides PROXY protocol support to Kengine by implementing the kengine.ListenerWrapper interface.
// It must be loaded before the `tls` listener.
//
// Deprecated: This kengine module should be replaced by the included proxy_protocol listener in Kengine.
type Wrapper struct {
	policy proxyproto.PolicyFunc
}

func (Wrapper) KengineModule() kengine.ModuleInfo {
	return kengine.ModuleInfo{
		ID:  "kengine.listeners.proxy_protocol",
		New: func() kengine.Module { return new(Wrapper) },
	}
}

func (pp *Wrapper) Provision(ctx kengine.Context) error {
	pp.policy = func(upstream net.Addr) (proxyproto.Policy, error) {
		return proxyproto.REQUIRE, nil
	}
	return nil
}

func (pp *Wrapper) WrapListener(l net.Listener) net.Listener {
	pL := &proxyproto.Listener{Listener: l, Policy: pp.policy}

	return pL
}
