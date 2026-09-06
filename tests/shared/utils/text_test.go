package utils_test

import (
	"testing"

	"auth_service/shared/utils"

	"github.com/stretchr/testify/assert"
)

// The key of a scoped profile is `<org_uuid>:<MacroCase(name)>`, so `:` has to fall
// in the separator swap - otherwise the generated key is ambiguous with the scope
// separator itself and a name of "a:b" produces a key nobody can split back.
func TestMacroCase(t *testing.T) {
	tests := map[string]string{
		"Editor":            "EDITOR",
		"Gestão de Vendas":  "GESTAO_DE_VENDAS",
		"read-only viewer":  "READ_ONLY_VIEWER",
		"  duplo   espaço ": "DUPLO_ESPACO",
		"a:b":               "A_B",
		"Ação Corretiva":    "ACAO_CORRETIVA",
		"admin__2":          "ADMIN_2",
		"MACRO_JÁ_PRONTO":   "MACRO_JA_PRONTO",
	}

	for name, expected := range tests {
		assert.Equal(t, expected, utils.MacroCase(name), "MacroCase(%q)", name)
	}
}

// A name that reduces to nothing is a 400 in the service, never a key ending in `:`.
func TestMacroCaseOfNothingIsEmpty(t *testing.T) {
	assert.Equal(t, "", utils.MacroCase("!!!"))
	assert.Equal(t, "", utils.MacroCase("   "))
	assert.Equal(t, "", utils.MacroCase(""))
}
