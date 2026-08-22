package parser

import (
	"strings"
	"testing"
)

func TestParseTypedConversionPredicates(t *testing.T) {
	checks := []struct {
		expression string
		predicate  string
	}{
		{`//div[string()='  keep  ']`, `string()='  keep  '`},
		{`//div[string(@empty)='']`, `string(@empty)=''`},
		{`//div[number()=1]`, `number()=1`},
		{`//div[number(@value)!=number('bad')]`, `number(@value)!=number('bad')`},
		{`//div[boolean(@empty) and not(boolean(@missing))]`, `boolean(@empty) and not (boolean(@missing))`},
		{`//div[string(number(' 01 '))='1']`, `string(number(' 01 '))='1'`},
		{`//div[not('01'='1') and @code=1]`, `not('01'='1') and @code=1`},
		{`//main[string(//a | //z)='first']`, `string(//a | //z)='first'`},
		{`//main[number(string((//a | //b)))=1]`, `number(string((//a | //b)))=1`},
		{`//case[left < right]`, `left < right`},
	}

	for _, check := range checks {
		parsed, err := NewParser().Parse(check.expression)
		if err != nil {
			t.Fatalf("Parse(%q): %v", check.expression, err)
		}
		if len(parsed.Steps) == 0 {
			t.Fatalf("Parse(%q) returned no steps", check.expression)
		}
		step := parsed.Steps[len(parsed.Steps)-1]
		if len(step.Predicates) != 1 || normalizeTypedPredicate(step.Predicates[0].Expression) != normalizeTypedPredicate(check.predicate) {
			t.Fatalf("Parse(%q) predicate = %#v, want %q", check.expression, step.Predicates, check.predicate)
		}
	}
}

func normalizeTypedPredicate(expression string) string {
	expression = strings.ReplaceAll(expression, "not (", "not(")
	return strings.ReplaceAll(expression, " | ", "|")
}

func TestParseRejectsInvalidTypedConversionArity(t *testing.T) {
	for _, expression := range []string{
		`//div[string('a', 'b')]`,
		`//div[number(1, 2)]`,
		`//div[boolean()]`,
		`//div[boolean(true(), false())]`,
	} {
		if _, err := NewParser().Parse(expression); err == nil {
			t.Errorf("Parse(%q) succeeded, want XPath function arity error", expression)
		}
	}
}

func TestParseNumericLiteralsWithLeadingDots(t *testing.T) {
	for _, expression := range []string{
		`//item[.5 < 1]`,
		`//item[.5]`,
	} {
		if _, err := NewParser().Parse(expression); err != nil {
			t.Errorf("Parse(%q): %v", expression, err)
		}
	}
	if _, err := NewParser().Parse(`//item[1.2.3]`); err == nil {
		t.Error("Parse accepted malformed numeric literal")
	}
}
