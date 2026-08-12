package bytepatch

import (
	"strings"
	"testing"
)

func TestHasBackrefs(t *testing.T) {
	cases := map[string]bool{
		`foo\1bar`:     true,
		`foo\\1bar`:    false, // escaped backslash + digit
		`no backrefs`:  false,
		`group (x) \2`: true,
	}
	for pat, want := range cases {
		if got := HasBackrefs(pat); got != want {
			t.Errorf("HasBackrefs(%q) = %v, want %v", pat, got, want)
		}
	}
}

func TestFindSubmatchBackrefsConsistent(t *testing.T) {
	// Mirrors the js-rule-deny-allow patch shape.
	pat := `if\(([A-Za-z_$][\w$]*)\?\.type==="rule"\)return\{behavior:"deny",message:([A-Za-z_$][\w$]*),decisionReason:\1\}`

	good := []byte(`if(a?.type==="rule")return{behavior:"deny",message:m,decisionReason:a}`)
	loc, re := FindSubmatchBackrefs(pat, good)
	if loc == nil {
		t.Fatal("expected verified match for consistent backref")
	}
	if re == nil {
		t.Fatal("expected concrete regexp for Expand")
	}
	// group 1 must be "a"
	if string(good[loc[2]:loc[3]]) != "a" {
		t.Fatalf("group 1 = %q, want a", good[loc[2]:loc[3]])
	}

	bad := []byte(`if(a?.type==="rule")return{behavior:"deny",message:m,decisionReason:b}`)
	if loc, _ := FindSubmatchBackrefs(pat, bad); loc != nil {
		t.Fatal("expected mismatch when backref differs from group 1")
	}
}

func TestFindSubmatchBackrefsSecondGroup(t *testing.T) {
	pat := `(x)(y)\2`
	if loc, _ := FindSubmatchBackrefs(pat, []byte("xyy")); loc == nil {
		t.Fatal("expected match xyy")
	}
	if loc, _ := FindSubmatchBackrefs(pat, []byte("xyx")); loc != nil {
		t.Fatal("xyx should not match (backref \\2 must equal group 2 = y)")
	}
}

func TestFindSubmatchBackrefsExpandRoundTrip(t *testing.T) {
	pat := `if\(([A-Za-z_$][\w$]*)\?\.type==="rule"\)return\{behavior:"deny",message:([A-Za-z_$][\w$]*),decisionReason:\1\}`
	data := []byte(`x;if(v?.type==="rule")return{behavior:"deny",message:msg,decisionReason:v};y`)
	loc, re := FindSubmatchBackrefs(pat, data)
	if loc == nil {
		t.Fatal("no match")
	}
	repl := re.Expand(nil, []byte(TranslateReplace(`if(\1?.type==="rule")return{behavior:"allow",message:\2,decisionReason:\1}`)), data, loc)
	want := `if(v?.type==="rule")return{behavior:"allow",message:msg,decisionReason:v}`
	if string(repl) != want {
		t.Fatalf("expand:\n got %s\nwant %s", repl, want)
	}
}

func TestFindSubmatchBackrefsNoBackrefPassthrough(t *testing.T) {
	loc, re := FindSubmatchBackrefs(`behavior:"deny"`, []byte(`a behavior:"deny" b`))
	if loc == nil || re == nil {
		t.Fatal("plain pattern should match")
	}
}

func TestTranslateReplace(t *testing.T) {
	in := `if(\1?.type==="rule")return{behavior:"allow",message:\2,decisionReason:\1}`
	out := TranslateReplace(in)
	if strings.Contains(out, `\1`) || strings.Contains(out, `\2`) {
		t.Fatalf("backrefs remain in replacement: %s", out)
	}
	want := `if($1?.type==="rule")return{behavior:"allow",message:$2,decisionReason:$1}`
	if out != want {
		t.Fatalf("TranslateReplace:\n got %s\nwant %s", out, want)
	}
}
