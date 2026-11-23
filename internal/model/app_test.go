package model

import "testing"

func TestRequestsProxyAndAIProxyRespectScopes(t *testing.T) {
	proxy := &Proxy{Host: "proxy.local", Port: 8080, Protocol: "http"}

	off := DefaultAppSettings()
	off.Proxy = proxy
	if off.RequestsProxy() != nil || off.AIProxy() != nil {
		t.Fatal("proxy must stay off while the master switch is disabled")
	}

	on := DefaultAppSettings()
	on.Proxy = proxy
	on.ProxyEnabled = true
	if on.RequestsProxy() == nil {
		t.Error("API requests should use the global proxy by default")
	}
	if on.AIProxy() == nil {
		t.Error("AI requests should use the global proxy by default")
	}

	aiOnly := on
	aiOnly.ProxyForRequests = false
	if aiOnly.RequestsProxy() != nil {
		t.Error("disabled request scope must not use the proxy")
	}
	if aiOnly.AIProxy() == nil {
		t.Error("AI scope should still use the proxy")
	}

	hostless := on
	hostless.Proxy = &Proxy{Port: 8080}
	if hostless.RequestsProxy() != nil || hostless.AIProxy() != nil {
		t.Error("a proxy without a host must be ignored")
	}
}
