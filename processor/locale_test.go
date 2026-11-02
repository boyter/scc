package processor

import "testing"

// pinLocale forces the English number formatting that the formatter
// expectations are written against.
//
// scc renders numbers through golang.org/x/text/message using the value of
// $LANG (see formatters_cost.go and formatters_tabular.go), so on any locale
// that formats numbers differently - for example es_ES.UTF-8, de_DE.UTF-8 or
// fr_FR.UTF-8 - the assertions below fail on the decimal separator and the
// thousands grouping.
func pinLocale(t *testing.T) {
	t.Helper()
	t.Setenv("LANG", "en_US.UTF-8")
}
