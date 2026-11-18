package execution

import (
	"context"
	"fmt"
	"strings"
)

// EgressProxyName é o nome padrão do contêiner do proxy de saída.
const EgressProxyName = "lf-egress-proxy"

// EgressConfig configura o proxy de saída compartilhado dos terminais dos agentes.
type EgressConfig struct {
	Name  string   // padrão EgressProxyName
	Image string   // imagem do proxy (make egress-image)
	Allow []string // destinos permitidos ("api.anthropic.com", ".openai.com")
	Ports []string // portas permitidas nos destinos; vazio = 443 e 80
}

// EnsureEgressProxy garante o contêiner do proxy de saída rodando com a lista de destinos pedida: reaproveita o que já
// roda com a mesma lista e recria se a lista mudou. O proxy fica na rede padrão do Docker (a única com saída); os
// sandboxes o alcançam por redes internas próprias (NetProxied). Em sucesso, d.EgressProxy aponta para ele.
func (d *Docker) EnsureEgressProxy(ctx context.Context, cfg EgressConfig) error {
	name := cfg.Name
	if name == "" {
		name = EgressProxyName
	}
	if cfg.Image == "" || len(cfg.Allow) == 0 {
		return fmt.Errorf("proxy de saída: informe a imagem e a lista de destinos permitidos")
	}
	allow := strings.Join(cfg.Allow, ",")
	ports := strings.Join(cfg.Ports, ",")
	fingerprint := allow + "#" + ports
	if out, _, err := d.run(ctx, nil, "inspect", "-f", `{{.State.Running}}|{{index .Config.Labels "lf.egress.allow"}}`, name); err == nil {
		if strings.TrimSpace(out) == "true|"+fingerprint {
			d.EgressProxy = name
			return nil
		}
		// a lista mudou (ou o contêiner parou): recria
		_, _, _ = d.run(ctx, nil, "rm", "-f", name)
	}
	if _, e, err := d.run(ctx, nil, "run", "-d", "--name", name, "--restart", "unless-stopped",
		"--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true", "--memory", "128m", "--pids-limit", "64",
		"--user", "65534:65534", "--label", "lf.egress.allow="+fingerprint, "-e", "EGRESS_ALLOW="+allow, "-e", "EGRESS_PORTS="+ports, cfg.Image); err != nil {
		return fmt.Errorf("subir o proxy de saída (%s): %s", cfg.Image, strings.TrimSpace(e))
	}
	d.EgressProxy = name
	return nil
}

// EgressProxyRunning diz se o contêiner do proxy está rodando.
func (d *Docker) EgressProxyRunning(ctx context.Context) bool {
	if d.EgressProxy == "" {
		return false
	}
	out, _, err := d.run(ctx, nil, "inspect", "-f", "{{.State.Running}}", d.EgressProxy)
	return err == nil && strings.TrimSpace(out) == "true"
}

// ProxyEnv é o ambiente que aponta o cliente do terminal para o proxy de saída.
func ProxyEnv() map[string]string {
	url := "http://" + EgressAlias + ":" + EgressPort
	return map[string]string{
		"HTTPS_PROXY": url, "HTTP_PROXY": url, "https_proxy": url, "http_proxy": url,
		"NO_PROXY": "localhost,127.0.0.1", "no_proxy": "localhost,127.0.0.1",
		"NODE_USE_ENV_PROXY": "1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
	}
}
