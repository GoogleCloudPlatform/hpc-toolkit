// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/pkg/errors"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// Reference is data struct that represents a reference to a variable.
// Neither checks are performed, nor context is captured, just a structural
// representation of a reference text
type Reference struct {
	GlobalVar bool
	Module    ModuleID // should be empty if GlobalVar. otherwise required
	Name      string   // required
}

// GlobalRef returns a reference to a global variable
func GlobalRef(n string) Reference {
	return Reference{GlobalVar: true, Name: n}
}

// ModuleRef returns a reference to a module output
func ModuleRef(m ModuleID, n string) Reference {
	return Reference{Module: m, Name: n}
}

// AsExpression returns a expression that represents the reference
func (r Reference) AsExpression() Expression {
	if r.GlobalVar {
		return MustParseExpression(fmt.Sprintf("var.%s", r.Name))
	}
	return MustParseExpression(fmt.Sprintf("module.%s.%s", r.Module, r.Name))
}

// Takes traversal in "blueprint namespace" (e.g. `vars.zone` or `homefs.mount`)
// and transforms it to `Expression`.
func bpTraversalToExpression(t hcl.Traversal) (Expression, error) {
	if len(t) < 2 {
		return nil, fmt.Errorf(expectedVarFormat)
	}
	attr, ok := t[1].(hcl.TraverseAttr)
	if !ok {
		return nil, fmt.Errorf(expectedVarFormat)
	}

	var ref Reference
	if t.RootName() == "vars" {
		t[0] = hcl.TraverseRoot{Name: "var"}
		ref = GlobalRef(attr.Name)
	} else {
		mod := t.RootName()
		t[0] = hcl.TraverseAttr{Name: mod}
		root := hcl.TraverseRoot{Name: "module"}
		t = append(hcl.Traversal{root}, t...)
		ref = ModuleRef(ModuleID(mod), attr.Name)
	}

	return &BaseExpression{
		e:    &hclsyntax.ScopeTraversalExpr{Traversal: t},
		toks: hclwrite.TokensForTraversal(t),
		rs:   []Reference{ref},
	}, nil
}

// bpLitToExpression takes a content of `$(...)`-literal and transforms it to `Expression`
func bpLitToExpression(s string) (Expression, error) {
	hexp, diag := hclsyntax.ParseExpression([]byte(s), "", hcl.Pos{})
	if diag.HasErrors() {
		return nil, diag
	}

	switch texp := hexp.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		exp, err := bpTraversalToExpression(texp.Traversal)
		if err != nil {
			return nil, fmt.Errorf("failed to parse variable %q: %w", s, err)
		}
		return exp, nil
	default:
		return nil, fmt.Errorf("only traversal expressions are supported, got %q", s)
	}
}

// TraversalToReference takes HCL traversal and returns `Reference`
func TraversalToReference(t hcl.Traversal) (Reference, error) {
	if t.IsRelative() {
		return Reference{}, fmt.Errorf("got relative traversal")
	}
	getAttrName := func(i int) (string, error) {
		if i >= len(t) {
			return "", fmt.Errorf("traversal does not have enough components")
		}
		if a, ok := t[i].(hcl.TraverseAttr); ok {
			return a.Name, nil
		}
		return "", fmt.Errorf("got unexpected traversal component: %#v", t[i])
	}
	switch root := t.RootName(); root {
	case "var":
		n, err := getAttrName(1)
		if err != nil {
			return Reference{}, fmt.Errorf("expected second component of global var reference to be a variable name, got %w", err)
		}
		return GlobalRef(n), nil
	case "module":
		m, err := getAttrName(1)
		if err != nil {
			return Reference{}, fmt.Errorf("expected second component of module var reference to be a module name, got %w", err)
		}
		n, err := getAttrName(2)
		if err != nil {
			return Reference{}, fmt.Errorf("expected third component of module var reference to be a variable name, got %w", err)
		}
		return ModuleRef(ModuleID(m), n), nil
	default:
		return Reference{}, fmt.Errorf("unexpected first component of reference: %#v", root)
	}
}

// Expression is a representation of expressions in Blueprint
type Expression interface {
	// Eval evaluates the expression in the context of Blueprint
	Eval(bp Blueprint) (cty.Value, error)
	// Tokenize returns Tokens to be used for marshalling HCL
	Tokenize() hclwrite.Tokens
	// References return Reference for all variables used in the expression
	References() []Reference
	// AsValue returns a cty.Value that represents the expression.
	// This function is the ONLY way to get an Expression as a cty.Value,
	// do not attempt to build it by other means.
	AsValue() cty.Value
	// key returns unique identifier of this expression in universe of all possible expressions.
	// `ex1.key() == ex2.key()` => `ex1` and `ex2` are identical.
	key() expressionKey
}

// ParseExpression returns Expression
// Expects expression in "terraform namespace" (e.g. `var.zone` or `module.homefs.mount`)
func ParseExpression(s string) (Expression, error) {
	e, diag := hclsyntax.ParseExpression([]byte(s), "", hcl.Pos{})
	if diag.HasErrors() {
		return nil, diag
	}
	sToks, _ := hclsyntax.LexExpression([]byte(s), "", hcl.Pos{})
	wToks := make(hclwrite.Tokens, len(sToks))
	for i, st := range sToks {
		wToks[i] = &hclwrite.Token{Type: st.Type, Bytes: st.Bytes}
	}

	ts := e.Variables()
	rs := make([]Reference, len(ts))
	for i, t := range ts {
		var err error
		if rs[i], err = TraversalToReference(t); err != nil {
			return nil, err
		}
	}
	return BaseExpression{e: e, toks: wToks, rs: rs}, nil
}

// MustParseExpression is "errorless" version of ParseExpression
// NOTE: only use it if passed expression is guaranteed to be correct
func MustParseExpression(s string) Expression {
	if exp, err := ParseExpression(s); err != nil {
		panic(fmt.Errorf("error while parsing %#v: %w", s, err))
	} else {
		return exp
	}
}

// BaseExpression is a base implementation of Expression interface
type BaseExpression struct {
	// Those fields should be accessed by Expression methods ONLY.
	e    hclsyntax.Expression
	toks hclwrite.Tokens
	rs   []Reference
}

// Eval evaluates the expression in the context of Blueprint
func (e BaseExpression) Eval(bp Blueprint) (cty.Value, error) {
	ctx := hcl.EvalContext{
		Variables: map[string]cty.Value{"var": bp.Vars.AsObject()},
		Functions: functions(),
	}
	v, diag := e.e.Value(&ctx)
	if diag.HasErrors() {
		return cty.NilVal, diag
	}
	return v, nil
}

// Tokenize returns Tokens to be used for marshalling HCL
func (e BaseExpression) Tokenize() hclwrite.Tokens {
	return e.toks
}

// References return Reference for all variables used in the expression
func (e BaseExpression) References() []Reference {
	c := make([]Reference, len(e.rs))
	copy(c, e.rs)
	return c
}

// key returns unique identifier of this expression in universe of all possible expressions.
// `ex1.key() == ex2.key()` => `ex1` and `ex2` are identical.
func (e BaseExpression) key() expressionKey {
	s := string(e.Tokenize().Bytes())
	return expressionKey{k: s}
}

// AsValue returns a cty.Value that represents the expression.
// This function is the ONLY way to get an Expression as a cty.Value,
// do not attempt to build it by other means.
func (e BaseExpression) AsValue() cty.Value {
	k := e.key()
	// we don't care if it overrides as expressions are identical
	globalExpressions[k] = e
	return cty.DynamicVal.Mark(k)
}

// To associate cty.Value with Expression we use cty.Value.Mark
// See: https://pkg.go.dev/github.com/zclconf/go-cty/cty#Value.Mark
// "Marks" should be of hashable type, sadly Expression isn't one.
// We work it around by using `expressionKey` - unique identifier
// of Expression and global map of used expressions `globalExpressions`.
// There are two guarantees to hold:
// * `ex1.key() == ex2.key()` => `ex1` and `ex2` are identical.
// It's achieved by using HCL literal as a key.
// * Every "mark" is in agreement with `globalExpressions`.
// Achieved by declaring `Expression.AsValue()` as the ONLY way to produce "marks".
type expressionKey struct {
	k string
}

var globalExpressions = map[expressionKey]Expression{}

// IsExpressionValue checks if the value is result of `Expression.AsValue()`.
// Returns original expression and result of check.
// It will panic if the value is expression-marked but not a result of `Expression.AsValue()`
func IsExpressionValue(v cty.Value) (Expression, bool) {
	key, ok := HasMark[expressionKey](v)
	if !ok {
		return nil, false
	}
	expr, stored := globalExpressions[key]
	if !stored { // shouldn't happen
		panic(fmt.Errorf("Expression isn't present in global state, while being referenced by value %#v", v))
	}
	return expr, true
}

// HasMark checks if cty.Value has mark of specified type T.
// Returns found mark and result of check.
// Panics if value has multiple marks of such type.
func HasMark[T any](val cty.Value) (T, bool) {
	var tgt T
	found := false
	for m := range val.Marks() {
		t, ok := m.(T)
		if !ok {
			continue
		}
		if found { // shouldn't happen
			panic(fmt.Errorf("more than one %T mark at %#v", tgt, val))
		}
		found, tgt = true, t
	}
	return tgt, found
}

// TokensForValue is a modification of hclwrite.TokensForValue.
// The only difference in behavior is handling "HCL literal" strings.
func TokensForValue(val cty.Value) hclwrite.Tokens {
	if val.IsNull() { // terminate early as Null value can has any type (e.g. String)
		return hclwrite.TokensForValue(val)
	}

	// We need to handle both cases, until all "expression" users are moved to Expression
	if e, is := IsExpressionValue(val); is {
		return e.Tokenize()
	}
	val, _ = val.Unmark() // remove marks, as we don't need them anymore
	ty := val.Type()
	if ty.IsListType() || ty.IsSetType() || ty.IsTupleType() {
		tl := []hclwrite.Tokens{}
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			tl = append(tl, TokensForValue(v))
		}
		return hclwrite.TokensForTuple(tl)
	}
	if ty.IsMapType() || ty.IsObjectType() {
		tl := []hclwrite.ObjectAttrTokens{}
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			kt := hclwrite.TokensForIdentifier(k.AsString())
			if !hclsyntax.ValidIdentifier(k.AsString()) {
				kt = TokensForValue(k)
			}
			vt := TokensForValue(v)
			tl = append(tl, hclwrite.ObjectAttrTokens{Name: kt, Value: vt})
		}
		return hclwrite.TokensForObject(tl)

	}
	return hclwrite.TokensForValue(val) // rely on hclwrite implementation
}

// FunctionCallExpression is a helper to build function call expression.
func FunctionCallExpression(n string, args ...cty.Value) Expression {
	if _, ok := functions()[n]; !ok {
		panic("unknown function " + n)
	}
	ta := make([]hclwrite.Tokens, len(args))
	for i, a := range args {
		ta[i] = TokensForValue(a)
	}
	toks := hclwrite.TokensForFunctionCall(n, ta...)
	return MustParseExpression(string(toks.Bytes()))
}

func functions() map[string]function.Function {
	return map[string]function.Function{
		"flatten": stdlib.FlattenFunc,
		"merge":   stdlib.MergeFunc,
	}
}

func valueReferences(v cty.Value) map[Reference]cty.Path {
	r := map[Reference]cty.Path{}
	cty.Walk(v, func(p cty.Path, v cty.Value) (bool, error) {
		if e, is := IsExpressionValue(v); is {
			for _, ref := range e.References() {
				r[ref] = p
			}
		}
		return true, nil
	})
	return r
}

func evalValue(v cty.Value, bp Blueprint) (cty.Value, error) {
	return cty.Transform(v, func(p cty.Path, v cty.Value) (cty.Value, error) {
		if e, is := IsExpressionValue(v); is {
			return e.Eval(bp)
		}
		return v, nil
	})
}

type pToken struct {
	s string
	e Expression
}

func tokenizeBpString(s string) ([]pToken, error) {
	toks := []pToken{}
	var exp Expression
	var err error
	bsRe := regexp.MustCompile(`\\*$`) // to count number of backslashes at the end

	for len(s) > 0 {
		i := strings.Index(s, "$(")
		if i == -1 { // plain string until the end
			toks, s = append(toks, pToken{s: s}), "" // add everything
			break                                    // and terminate
		}
		p := s[:i]
		s = s[i+2:]                       // split as `p$(s`
		bs := len(bsRe.FindString(p))     // get number of trailing backslashes
		p = p[:len(p)-bs+bs/2]            // keep (smaller) half of backslashes
		toks = append(toks, pToken{s: p}) // add tokens up to "$("

		if bs%2 == 1 { // escaped $(
			toks = append(toks, pToken{s: "$("}) // add "$("
		} else { // found beginning of expression
			exp, s, err = greedyParseHcl(s) // parse after "$("
			if err != nil {
				return nil, err
			}
			toks = append(toks, pToken{e: exp}) // add expression
		}
	}
	return toks, nil
}

func compactTokens(toks []pToken) []pToken {
	res := []pToken{}
	for _, t := range toks {
		if t.e != nil {
			res = append(res, t) // add as is
		} else {
			if t.s == "" {
				continue // skip
			}
			if len(res) > 0 && res[len(res)-1].e == nil {
				res[len(res)-1].s += t.s // merge with previous
			} else {
				res = append(res, t) // add as is
			}
		}
	}
	return res
}

func parseBpLit(s string) (cty.Value, error) {
	toks, err := tokenizeBpString(s)
	if err != nil {
		return cty.NilVal, err
	}
	toks = compactTokens(toks)
	if len(toks) == 0 {
		return cty.StringVal(""), nil
	}
	if len(toks) == 1 {
		if toks[0].e != nil {
			return toks[0].e.AsValue(), nil
		} else {
			return cty.StringVal(toks[0].s), nil
		}
	}

	exp, err := buildStringInterpolation(toks)
	if err != nil {
		return cty.NilVal, err
	}
	return exp.AsValue(), nil
}

// greedyParseHcl tries to parse prefix of `s` as a valid HCL expression.
// It iterates over all closing brackets and tries to parse expression up to them.
// The shortest expression is returned. E.g:
// "var.hi) $(var.there)" -> "var.hi"
// "try(var.this) + one(var.time)) tail" -> "try(var.this) + one(var.time)"
func greedyParseHcl(s string) (Expression, string, error) {
	err := errors.New("no closing parenthesis")
	for i := 0; i < len(s); i++ {
		if s[i] != ')' {
			continue
		}
		_, diag := hclsyntax.ParseExpression([]byte(s[:i]), "", hcl.Pos{})
		if !diag.HasErrors() { // found an expression
			exp, err := bpLitToExpression(s[:i])
			return exp, s[i+1:], err
		}
		err = diag // save error, try to find another closing bracket
	}
	return nil, s, err
}

func buildStringInterpolation(pts []pToken) (Expression, error) {
	toks := hclwrite.Tokens{&hclwrite.Token{
		Type:  hclsyntax.TokenOQuote,
		Bytes: []byte(`"`)},
	}

	for _, pt := range pts {
		if pt.e != nil {
			toks = append(toks, &hclwrite.Token{
				Type:  hclsyntax.TokenTemplateInterp,
				Bytes: []byte(`${`)})
			toks = append(toks, pt.e.Tokenize()...)
			toks = append(toks, &hclwrite.Token{
				Type:  hclsyntax.TokenTemplateSeqEnd,
				Bytes: []byte(`}`)})
		} else {
			stoks := hclwrite.TokensForValue(cty.StringVal(pt.s))
			stoks = stoks[1 : len(stoks)-1] // remove quotes
			toks = append(toks, stoks...)
		}
	}

	toks = append(toks, &hclwrite.Token{
		Type:  hclsyntax.TokenCQuote,
		Bytes: []byte(`"`)})
	return ParseExpression(string(toks.Bytes()))
}
