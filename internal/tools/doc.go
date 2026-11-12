// Package tools mantém o catálogo de ferramentas (nativas e personalizadas) e
// aplica, antes de qualquer efeito, a ordem: ferramenta existe → foi concedida
// ao agente → o escopo (permissões) cobre → os argumentos batem com o esquema →
// a política autoriza (T08-03). A saída volta tipada conforme o esquema.
package tools
