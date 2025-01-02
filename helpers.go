package main

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"unicode"
)

func toSnakeCase(s string) string {
	match := regexp.MustCompilePOSIX("([a-z])([A-Z]|[0-9])|[0-9][A-Z]")
	return match.ReplaceAllString(s, "${1}_${2}")
}

func setUnexported(name string) string {
	r := []rune(name)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func setExported(name string) string {
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func addModifiedComment(file *ast.File) {
	file.Comments[0].List = append(file.Comments[0].List, &ast.Comment{
		Text: "// Modified by sqlc-metrics-generator v1.0.0",
	})
}

func previouslyModified(file *ast.File) bool {
	for _, comment := range file.Comments {
		for _, c := range comment.List {
			if strings.Contains(c.Text, "sqlc-metrics-generator") {
				return true
			}
		}
	}
	return false
}

func addMissingImports(file *ast.File, imports []string) {
	requiredImports := map[string]bool{}
	for _, imp := range imports {
		requiredImports[imp] = true
	}
	for _, decl := range file.Decls {
		if GenDecl, ok := decl.(*ast.GenDecl); ok && GenDecl.Tok == token.IMPORT {
			for _, spec := range GenDecl.Specs {
				if BasicLit, ok := spec.(*ast.ImportSpec); ok {
					if _, ok := requiredImports[strings.ReplaceAll(BasicLit.Path.Value, "\"", "")]; ok {
						delete(requiredImports, strings.ReplaceAll(BasicLit.Path.Value, "\"", ""))
					}
				}
			}
		}
	}
	for i, decl := range file.Decls {
		if GenDecl, ok := decl.(*ast.GenDecl); ok && GenDecl.Tok == token.IMPORT {
			for imp, _ := range requiredImports {
				file.Decls[i].(*ast.GenDecl).Specs = append(file.Decls[i].(*ast.GenDecl).Specs, &ast.ImportSpec{
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: "\"" + imp + "\"",
					},
				},
				)
			}
		}
	}
}
