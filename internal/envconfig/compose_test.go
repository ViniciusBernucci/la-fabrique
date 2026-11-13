package envconfig

import (
	"strings"
	"testing"
)

func TestComposePolicyRejectsEscapes(t *testing.T) {
	for name, body := range map[string]string{
		"privileged": "services:\n  a:\n    image: x\n    privileged: true\n",
		"host net":   "services:\n  a:\n    image: x\n    network_mode: host\n",
		"socket":     "services:\n  a:\n    image: x\n    volumes:\n      - /var/run/docker.sock:/var/run/docker.sock\n",
		"abs bind":   "services:\n  a:\n    image: x\n    volumes:\n      - /etc:/data\n",
		"parent":     "services:\n  a:\n    image: x\n    volumes:\n      - ../segredos:/s\n",
		"name":       "services:\n  a:\n    image: x\n    container_name: fixo\n",
		"cap":        "services:\n  a:\n    image: x\n    cap_add:\n      - SYS_ADMIN\n",
	} {
		if len(CheckCompose(body)) == 0 {
			t.Errorf("%s deveria ser recusada", name)
		}
	}
	ok := "services:\n  web:\n    image: x\n    volumes:\n      - ./src:/app/src\n      - dados:/var/lib\n    ports:\n      - \"5180:5173\"\n"
	if p := CheckCompose(ok); len(p) != 0 {
		t.Fatalf("composição comum deve passar: %v", p)
	}
}

func TestOverrideRemovesPublishedPortsMakesTheNetworkInternalAndLabels(t *testing.T) {
	o := Override([]string{"web", "db"}, "e1", "lfenv-e1")
	for _, want := range []string{"ports: !reset []", "internal: true", `lf.exec: "e1"`, "  web:", "  db:"} {
		if !strings.Contains(o, want) {
			t.Fatalf("falta %q:\n%s", want, o)
		}
	}
}
