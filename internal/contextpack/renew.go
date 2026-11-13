package contextpack

import "fmt"

// Métodos de medição do uso da sessão.
const (
	UsageOfficial = "official"
	UsageEstimate = "estimate"
	UsageNone     = "none"
)

// Usage é o que se sabe do consumo da sessão. Official é a métrica do
// fornecedor, quando existe; sem ela, Estimated é a estimativa local (tokens
// de tudo o que foi enviado e recebido).
type Usage struct {
	Official  *int64
	Estimated int64
	Window    int64 // janela de contexto do modelo
}

// Policy define quando renovar.
type Policy struct {
	Threshold      float64 // fração da janela; padrão 0,8
	EstimateMargin float64 // multiplicador sobre a estimativa; padrão 1,25
}

// Decision é o resultado auditável.
type Decision struct {
	Renew  bool    `json:"renew"`
	Method string  `json:"method"`
	Used   int64   `json:"used"`
	Window int64   `json:"window"`
	Ratio  float64 `json:"ratio"`
}

// Decide escolhe renovar antes de a janela saturar. Sem métrica oficial usa a
// estimativa com margem (conservadora). Sem janela conhecida, falha.
func Decide(u Usage, p Policy) (Decision, error) {
	if u.Window <= 0 {
		return Decision{}, &FieldError{"window", "informe a janela de contexto do modelo"}
	}
	if p.Threshold <= 0 || p.Threshold > 1 {
		p.Threshold = 0.8
	}
	if p.EstimateMargin < 1 {
		p.EstimateMargin = 1.25
	}
	d := Decision{Window: u.Window}
	if u.Official != nil {
		if *u.Official < 0 {
			return Decision{}, &FieldError{"official", "uso oficial negativo"}
		}
		d.Method, d.Used = UsageOfficial, *u.Official
	} else {
		if u.Estimated < 0 {
			return Decision{}, &FieldError{"estimated", "estimativa negativa"}
		}
		d.Method, d.Used = UsageEstimate, int64(float64(u.Estimated)*p.EstimateMargin+0.999999)
	}
	d.Ratio = float64(d.Used) / float64(u.Window)
	d.Renew = d.Ratio >= p.Threshold
	return d, nil
}

func (d Decision) String() string {
	return fmt.Sprintf("%s: %d/%d (%.0f%%)", d.Method, d.Used, d.Window, d.Ratio*100)
}
