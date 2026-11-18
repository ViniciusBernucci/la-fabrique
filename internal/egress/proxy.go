// Package egress é o proxy de saída dos terminais dos agentes (ADR-0069). O terminal roda numa rede Docker interna,
// sem rota para a internet; o único caminho de saída é este proxy, que só deixa passar os destinos da lista
// (os serviços da conta de IA, por exemplo). Isso torna seguro o que o agente executa dentro do terminal: um comando
// não consegue enviar dados a um destino fora da lista, nem por socket direto, porque não há rota.
package egress

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Proxy é um proxy HTTP/HTTPS (método CONNECT) com lista de destinos permitidos.
type Proxy struct {
	// Allow é a lista de destinos: "api.anthropic.com" (exato) ou ".openai.com" (esse domínio e subdomínios).
	Allow []string
	// Ports permitidos nos destinos; vazio = 443 e 80.
	Ports []string
	Log   *slog.Logger
	// Dial permite ensaios; padrão net.DialTimeout.
	Dial func(network, addr string, timeout time.Duration) (net.Conn, error)
}

func (p *Proxy) logger() *slog.Logger {
	if p.Log != nil {
		return p.Log
	}
	return slog.Default()
}

// Allowed diz se o destino (host sem porta) está na lista.
func (p *Proxy) Allowed(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, a := range p.Allow {
		a = strings.ToLower(strings.TrimSpace(a))
		switch {
		case a == "":
		case strings.HasPrefix(a, "."):
			if host == a[1:] || strings.HasSuffix(host, a) {
				return true
			}
		case host == a:
			return true
		}
	}
	return false
}

func (p *Proxy) portOK(port string) bool {
	ports := p.Ports
	if len(ports) == 0 {
		ports = []string{"443", "80"}
	}
	for _, x := range ports {
		if x == port {
			return true
		}
	}
	return false
}

func (p *Proxy) dial(addr string) (net.Conn, error) {
	d := p.Dial
	if d == nil {
		d = net.DialTimeout
	}
	return d("tcp", addr, 15*time.Second)
}

// ServeHTTP atende CONNECT (HTTPS) e requisições HTTP absolutas.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.connect(w, r)
		return
	}
	p.forward(w, r)
}

func (p *Proxy) deny(w http.ResponseWriter, host, why string) {
	p.logger().Warn("saída negada", "host", host, "motivo", why)
	http.Error(w, "destino não permitido pelo proxy de saída da fábrica: "+host, http.StatusForbidden)
}

func (p *Proxy) connect(w http.ResponseWriter, r *http.Request) {
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		http.Error(w, "destino inválido", http.StatusBadRequest)
		return
	}
	if !p.Allowed(host) {
		p.deny(w, host, "fora da lista")
		return
	}
	if !p.portOK(port) {
		p.deny(w, host+":"+port, "porta não permitida")
		return
	}
	up, err := p.dial(net.JoinHostPort(host, port))
	if err != nil {
		p.logger().Warn("destino inalcançável", "host", host, "erro", err.Error())
		http.Error(w, "destino inalcançável", http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = up.Close()
		http.Error(w, "proxy sem suporte a CONNECT", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	client, _, err := hj.Hijack()
	if err != nil {
		_ = up.Close()
		return
	}
	p.logger().Info("saída permitida", "host", host, "porta", port)
	var wg sync.WaitGroup
	wg.Add(2)
	pipe := func(dst, src net.Conn) {
		defer wg.Done()
		_, _ = io.Copy(dst, src)
		if c, ok := dst.(interface{ CloseWrite() error }); ok {
			_ = c.CloseWrite()
		}
	}
	go pipe(up, client)
	go pipe(client, up)
	wg.Wait()
	_ = client.Close()
	_ = up.Close()
}

func (p *Proxy) forward(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host == "" {
		http.Error(w, "requisição sem destino absoluto", http.StatusBadRequest)
		return
	}
	host := r.URL.Hostname()
	port := r.URL.Port()
	if port == "" {
		port = "80"
	}
	if !p.Allowed(host) {
		p.deny(w, host, "fora da lista")
		return
	}
	if !p.portOK(port) {
		p.deny(w, host+":"+port, "porta não permitida")
		return
	}
	out := r.Clone(context.Background())
	out.RequestURI = ""
	out.Header.Del("Proxy-Connection")
	tr := &http.Transport{DialContext: func(_ context.Context, network, addr string) (net.Conn, error) { return p.dial(addr) }}
	resp, err := tr.RoundTrip(out)
	if err != nil {
		http.Error(w, "destino inalcançável", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// ParseList separa uma lista "a,b , .c" em itens.
func ParseList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Describe resume a configuração para o log de partida.
func (p *Proxy) Describe() string {
	return fmt.Sprintf("%d destino(s) permitido(s): %s", len(p.Allow), strings.Join(p.Allow, ", "))
}
