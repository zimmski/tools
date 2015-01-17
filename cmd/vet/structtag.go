// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains the test for canonical struct tags.

package main

import (
	"errors"
	"go/ast"
	"reflect"
	"strconv"
	"unicode"
	"unicode/utf8"
)

func init() {
	register("structtags",
		"check that struct field tags have canonical format and apply to exported fields as needed",
		checkCanonicalFieldTag,
		field)
}

// checkCanonicalFieldTag checks a struct field tag.
func checkCanonicalFieldTag(f *File, node ast.Node) {
	field := node.(*ast.Field)
	if field.Tag == nil {
		return
	}

	tag, err := strconv.Unquote(field.Tag.Value)
	if err != nil {
		f.Badf(field.Pos(), "unable to read struct tag %s", field.Tag.Value)
		return
	}

	if err := validateStructTag(tag); err != nil {
		f.Badf(field.Pos(), "struct field tag %s not compatible with reflect.StructTag.Get: %s", field.Tag.Value, err)
	}

	// Check for use of json or xml tags with unexported fields.

	// Embedded struct. Nothing to do for now, but that
	// may change, depending on what happens with issue 7363.
	if len(field.Names) == 0 {
		return
	}

	if field.Names[0].IsExported() {
		return
	}

	st := reflect.StructTag(tag)
	for _, enc := range [...]string{"json", "xml"} {
		if st.Get(enc) != "" {
			f.Badf(field.Pos(), "struct field %s has %s tag but is not exported", field.Names[0].Name, enc)
			return
		}
	}
}

var (
	errTagSyntax      = errors.New("bad syntax for struct tag pair")
	errTagKeySyntax   = errors.New("bad syntax for struct tag key")
	errTagValueSyntax = errors.New("bad syntax for struct tag value")
)

// validateStructTag parses the struct tag and returns an error if it is not
// in the canonical format, which is a space-separated list of key:"value"
// settings.
func validateStructTag(tag string) error {
	l := len(tag)

	for i := 0; i < l; i++ {
		// Ignore spaces
		for i < l && tag[i] == ' ' {
			i++
		}

		if i >= l {
			break
		}

		// Key must not contain control characters or quotes.
		j := i
		for ; i < l; i++ {
			if tag[i] == ':' {
				// Key must not be empty
				if i == j {
					return errTagSyntax
				}

				i++

				break
			} else if r, w := utf8.DecodeRuneInString(tag[i:]); tag[i] == '"' || unicode.IsControl(r) {
				return errTagKeySyntax
			} else {
				// Move i further along if we encountered an UTF8 character
				if w > 1 {
					i += w - 1
				}
			}
		}

		// There must be room for a quoted string
		if i > l-2 {
			return errTagSyntax
		}

		// Check the starting quote
		if tag[i] != '"' {
			return errTagValueSyntax
		}

		j = i

		// Jump over quote
		i++

		for i < l && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}

			i++
		}

		// Check if there was an ending quote
		if i >= l {
			return errTagValueSyntax
		}

		// Jump over quote
		i++

		_, err := strconv.Unquote(tag[j:i])
		if err != nil {
			return errTagValueSyntax
		}

		// There must be at least one space between two pairs
		if i < l && tag[i] != ' ' {
			return errTagSyntax
		}
	}

	return nil
}
