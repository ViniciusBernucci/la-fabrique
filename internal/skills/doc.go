// Package skills mantém o catálogo de skills e importa pacotes com segurança
// (T08-02). Pacotes são dados: a importação só lê, valida caminhos, tamanhos e
// scripts contra a política e NUNCA executa nada. A execução (IA, script,
// determinística ou híbrida) é um passo separado, sempre sob a política.
package skills
