package region

import "testing"

func TestResolveMappedCIDR(t *testing.T) {
	r := NewResolver("203.0.113.0/24=110105;198.51.100.0/24=310104")

	region, ok := r.Resolve("203.0.113.7")
	if !ok || region != "110105" {
		t.Fatalf("expected 110105, got %q (ok=%v)", region, ok)
	}

	region, ok = r.Resolve("198.51.100.200")
	if !ok || region != "310104" {
		t.Fatalf("expected 310104, got %q (ok=%v)", region, ok)
	}
}

func TestResolveUnknownIsNotGuessed(t *testing.T) {
	r := NewResolver("203.0.113.0/24=110105")

	// 未映射的地址必须返回未知，而不是编造或回退到请求自报值
	if _, ok := r.Resolve("8.8.8.8"); ok {
		t.Fatal("unmapped IP must not resolve to a region")
	}
	if _, ok := r.Resolve("10.0.0.1"); ok {
		t.Fatal("private IP must not resolve without explicit mapping")
	}
	if _, ok := r.Resolve("127.0.0.1"); ok {
		t.Fatal("loopback IP must not resolve without explicit mapping")
	}
}

func TestResolveRejectsGarbage(t *testing.T) {
	r := NewResolver("203.0.113.0/24=110105")

	for _, ip := range []string{"", "not-an-ip", "110105", "203.0.113.999"} {
		if _, ok := r.Resolve(ip); ok {
			t.Fatalf("garbage input %q must not resolve", ip)
		}
	}
}

func TestEmptyOrInvalidMapping(t *testing.T) {
	// 空配置与非法条目全部跳过，解析结果一律未知
	for _, mapping := range []string{"", ";;;", "bad-entry", "10.0.0.0/8=", "999.0.0.0/8=110105", "=110105"} {
		r := NewResolver(mapping)
		if _, ok := r.Resolve("10.1.2.3"); ok {
			t.Fatalf("mapping %q must yield no entries", mapping)
		}
	}
}

func TestIPv6Mapping(t *testing.T) {
	r := NewResolver("2001:db8::/32=110105")
	region, ok := r.Resolve("2001:db8::1")
	if !ok || region != "110105" {
		t.Fatalf("expected 110105, got %q (ok=%v)", region, ok)
	}
}
