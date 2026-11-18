package egress

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAllowedMatchesExactAndSubdomains(t *testing.T) {
	p := &Proxy{Allow: []string{"api.anthropic.com", ".openai.com", "  "}}
	for host, want := range map[string]bool{
		"api.anthropic.com": true, "API.Anthropic.com.": true, "x.api.anthropic.com": false, "anthropic.com": false,
		"auth.openai.com": true, "openai.com": true, "evilopenai.com": false, "openai.com.evil.io": false, "": false,
	} {
		if got := p.Allowed(host); got != want {
			t.Errorf("Allowed(%q) = %v, esperado %v", host, got, want)
		}
	}
}

// proxy de verdade, destino de mentira: CONNECT permitido atravessa; destino fora da lista e porta fora da lista são recusados.
func TestConnectOnlyReachesAllowedDestinations(t *testing.T) {
	up, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer up.Close()
	go func() {
		for {
			c, err := up.Accept()
			if err != nil {
				return
			}
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }() // eco
		}
	}()
	upHost, upPort, _ := net.SplitHostPort(up.Addr().String())

	p := &Proxy{Allow: []string{"permitido.example"}, Ports: []string{upPort},
		Dial: func(_, _ string, d time.Duration) (net.Conn, error) {
			return net.DialTimeout("tcp", up.Addr().String(), d)
		}}
	srv := httptest.NewServer(p)
	defer srv.Close()
	pu, _ := url.Parse(srv.URL)

	connect := func(target string) (int, net.Conn) {
		c, err := net.Dial("tcp", pu.Host)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(c, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
		br := bufio.NewReader(c)
		resp, err := http.ReadResponse(br, &http.Request{Method: "CONNECT"})
		if err != nil {
			t.Fatal(err)
		}
		return resp.StatusCode, c
	}
	code, c := connect("permitido.example:" + upPort)
	if code != 200 {
		t.Fatalf("destino permitido deveria passar: %d", code)
	}
	fmt.Fprint(c, "ola\n")
	buf := make([]byte, 4)
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.ReadFull(c, buf); err != nil || string(buf) != "ola\n" {
		t.Fatalf("o túnel deveria ecoar: %q %v", buf, err)
	}
	c.Close()
	if code, c2 := connect("proibido.example:" + upPort); code != http.StatusForbidden {
		t.Fatalf("destino fora da lista deveria ser 403: %d", code)
	} else {
		c2.Close()
	}
	if code, c3 := connect("permitido.example:22"); code != http.StatusForbidden {
		t.Fatalf("porta fora da lista deveria ser 403: %d", code)
	} else {
		c3.Close()
	}
	_ = upHost
}

func TestForwardAbsoluteHTTPAndDenial(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "origem ok") }))
	defer origin.Close()
	ou, _ := url.Parse(origin.URL)
	p := &Proxy{Allow: []string{ou.Hostname()}, Ports: []string{ou.Port()}}
	srv := httptest.NewServer(p)
	defer srv.Close()
	pu, _ := url.Parse(srv.URL)
	cl := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(pu)}, Timeout: 5 * time.Second}
	resp, err := cl.Get(origin.URL)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || string(b) != "origem ok" {
		t.Fatalf("encaminhamento HTTP: %d %q", resp.StatusCode, b)
	}
	denied := &Proxy{Allow: []string{"outro.example"}, Ports: []string{ou.Port()}}
	srv2 := httptest.NewServer(denied)
	defer srv2.Close()
	pu2, _ := url.Parse(srv2.URL)
	cl2 := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(pu2)}, Timeout: 5 * time.Second}
	r2, err := cl2.Get(origin.URL)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusForbidden {
		t.Fatalf("destino fora da lista deveria ser 403: %d", r2.StatusCode)
	}
}

func TestParseList(t *testing.T) {
	if got := strings.Join(ParseList(" a.com, .b.com ,, c "), "|"); got != "a.com|.b.com|c" {
		t.Fatal(got)
	}
}
