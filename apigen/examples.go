package apigen

import (
	"hash/fnv"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

// maxExampleDepth bounds recursion so self-referential structs can't loop forever.
const maxExampleDepth = 6

var timeType = reflect.TypeOf(time.Time{})

// exampleGen produces realistic, *deterministic* example values for documentation.
//
// Determinism matters: docs are regenerated on every build and committed/diffed,
// so the same struct must always yield the same example. We get realistic data
// from gofakeit but seed it from the struct's type name, so output is stable
// run-to-run while still varying between different types.
//
// Per-field generation is controlled, in order of precedence, by:
//
//	example:"<literal>"  -> used verbatim (coerced to the field's kind)
//	fake:"<template>"    -> gofakeit template, e.g. fake:"{email}" or "{number:1,99}"
//	field-name heuristics-> e.g. a string field named "Email" gets a real email
//	the field's type     -> sensible fallback per Go kind
type exampleGen struct {
	faker *gofakeit.Faker
}

// newExampleGen returns a generator seeded deterministically from seedKey
// (typically the struct's type name).
func newExampleGen(seedKey string) *exampleGen {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seedKey))
	return &exampleGen{faker: gofakeit.New(h.Sum64())}
}

// forField generates an example for a struct field, honoring its tags.
func (g *exampleGen) forField(field reflect.StructField, depth int) any {
	if lit, ok := field.Tag.Lookup("example"); ok {
		return coerceScalar(lit, field.Type)
	}
	if tmpl, ok := field.Tag.Lookup("fake"); ok && tmpl != "" && tmpl != "skip" && tmpl != "-" {
		if v, err := g.faker.Generate(tmpl); err == nil {
			return coerceScalar(v, field.Type)
		}
	}
	return g.forType(field.Type, fieldHint(field), depth)
}

// forType generates an example for a type. hint is a lowercased name used for
// string heuristics (e.g. "email", "createdat").
func (g *exampleGen) forType(t reflect.Type, hint string, depth int) any {
	if depth > maxExampleDepth {
		return nil
	}
	if t == timeType {
		return g.faker.Date().UTC().Format(time.RFC3339)
	}

	switch t.Kind() {
	case reflect.Ptr:
		return g.forType(t.Elem(), hint, depth)
	case reflect.Struct:
		m := make(map[string]any)
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			// Flatten anonymous embedded structs, matching extractStructFields.
			if f.Anonymous && deref(f.Type).Kind() == reflect.Struct {
				if sub, ok := g.forType(deref(f.Type), hint, depth+1).(map[string]any); ok {
					for k, v := range sub {
						m[k] = v
					}
				}
				continue
			}
			jn := getJSONFieldName(f)
			if jn == "" {
				continue
			}
			m[jn] = g.forField(f, depth+1)
		}
		return m
	case reflect.Slice, reflect.Array:
		return []any{g.forType(t.Elem(), singular(hint), depth+1)}
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			return map[string]any{g.faker.Word(): g.forType(t.Elem(), hint, depth+1)}
		}
		return map[string]any{}
	case reflect.String:
		return g.stringFor(hint)
	case reflect.Bool:
		return g.faker.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.faker.Number(1, 1000)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uint(g.faker.Number(1, 1000))
	case reflect.Float32, reflect.Float64:
		return float64(g.faker.Number(1, 100000)) / 100
	case reflect.Interface:
		return nil
	default:
		return nil
	}
}

// stringFor returns realistic content for a string field based on its name.
func (g *exampleGen) stringFor(hint string) string {
	switch {
	case contains(hint, "email"):
		return g.faker.Email()
	case contains(hint, "phone", "mobile", "contact"):
		return g.faker.Phone()
	case contains(hint, "firstname"):
		return g.faker.FirstName()
	case contains(hint, "lastname", "surname"):
		return g.faker.LastName()
	case contains(hint, "username", "handle"):
		return g.faker.Username()
	case contains(hint, "name"):
		return g.faker.Name()
	case contains(hint, "uuid"), hint == "id", strings.HasSuffix(hint, "id"):
		return g.faker.UUID()
	case contains(hint, "avatar", "image", "photo", "picture", "logo", "url", "uri", "link", "website"):
		return g.faker.URL()
	case contains(hint, "city"):
		return g.faker.City()
	case contains(hint, "country"):
		return g.faker.Country()
	case contains(hint, "state", "province"):
		return g.faker.State()
	case contains(hint, "zip", "postal"):
		return g.faker.Zip()
	case contains(hint, "address", "street"):
		return g.faker.Street()
	case contains(hint, "gender"):
		return g.faker.Gender()
	case contains(hint, "color", "colour"):
		return g.faker.Color()
	case contains(hint, "company", "organization", "org"):
		return g.faker.Company()
	case contains(hint, "currency"):
		return g.faker.CurrencyShort()
	case contains(hint, "date", "time", "at"):
		return g.faker.Date().UTC().Format(time.RFC3339)
	case contains(hint, "token", "jwt", "secret", "key", "hash", "password"):
		return g.faker.LetterN(32)
	case contains(hint, "description", "bio", "message", "comment", "content", "summary", "text", "note", "title"):
		return g.faker.Sentence(8)
	case contains(hint, "status", "type", "category", "role", "kind"):
		return g.faker.Word()
	case contains(hint, "slug"):
		return g.faker.Word()
	default:
		return g.faker.Word()
	}
}

// coerceScalar converts a string literal (from an example: tag or a fake
// template) into the field's underlying kind, so example:"42" on an int field
// yields the number 42 rather than the string "42".
func coerceScalar(s string, t reflect.Type) any {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		if b, err := strconv.ParseBool(s); err == nil {
			return b
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	case reflect.Float32, reflect.Float64:
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return s
}

// fieldHint returns a lowercased name (preferring the JSON name) for heuristics.
func fieldHint(f reflect.StructField) string {
	if jn := getJSONFieldName(f); jn != "" {
		return strings.ToLower(jn)
	}
	return strings.ToLower(f.Name)
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func contains(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// singular is a tiny best-effort de-pluralizer for slice element hints.
func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "ies") && len(s) > 3:
		return s[:len(s)-3] + "y"
	case strings.HasSuffix(s, "ses") && len(s) > 3:
		return s[:len(s)-2]
	case strings.HasSuffix(s, "s") && len(s) > 1:
		return s[:len(s)-1]
	default:
		return s
	}
}
