package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"strconv"
)

func modifyQuerySqlFile(file *ast.File, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics bool) (*ast.File, []string, error) {

	if previouslyModified(file) {
		panic("modifyQuerySqlFile: file has already been modified")
	}
	addModifiedComment(file)

	requiredImports := []string{
		"context",
		"go.opentelemetry.io/otel/attribute",
		"go.opentelemetry.io/otel/metric",
		"time",
	}
	var foundFunctions []string
	addMissingImports(file, requiredImports)

	var versions []ast.Decl
	for i, decl := range file.Decls {
		v, err := generateVersionConstants(decl)
		if err != nil {
			return nil, nil, err
		}
		if v != nil {
			versions = append(versions, v)
		}
		foundFunctions = addFoundFunction(decl, foundFunctions)
		renameAndWrap(file, &decl, i, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics)
	}
	file.Decls = append(file.Decls, versions...)
	return file, foundFunctions, nil
}

func addFoundFunction(decl ast.Decl, foundFunctions []string) []string {
	if FuncDecl, ok := decl.(*ast.FuncDecl); ok {
		foundFunctions = append(foundFunctions, FuncDecl.Name.Name)
	}
	return foundFunctions
}

func renameAndWrap(file *ast.File, decl *ast.Decl, i int, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics bool) {
	if FuncDecl, ok := (*decl).(*ast.FuncDecl); ok {
		name := FuncDecl.Name.Name
		file.Decls[i].(*ast.FuncDecl).Name.Name = setUnexported(FuncDecl.Name.Name) + "Original"
		var Stmt []ast.Stmt
		if generateQueryRuntimeMetrics {
			Stmt = append(Stmt, &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.Ident{
								Name: "startTime",
							},
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.CallExpr{
								Fun: &ast.SelectorExpr{
									X: &ast.Ident{
										Name: "time",
									},
									Sel: &ast.Ident{
										Name: "Now",
									},
								},
							},
						},
					},
					&ast.DeferStmt{
						Call: &ast.CallExpr{
							Fun: &ast.FuncLit{
								Type: &ast.FuncType{
									Params: &ast.FieldList{},
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X: &ast.SelectorExpr{
														X: &ast.Ident{
															Name: "q",
														},
														Sel: &ast.Ident{
															Name: setUnexported(name) + "RuntimeGauge",
														},
													},
													Sel: &ast.Ident{
														Name: "Record",
													},
												},
												Args: []ast.Expr{
													&ast.Ident{
														Name: "ctx",
													},
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X: &ast.CallExpr{
																Fun: &ast.SelectorExpr{
																	X: &ast.Ident{
																		Name: "time",
																	},
																	Sel: &ast.Ident{
																		Name: "Since",
																	},
																},
																Args: []ast.Expr{
																	&ast.Ident{
																		Name: "startTime",
																	},
																},
															},
															Sel: &ast.Ident{
																Name: "Seconds",
															},
														},
													},
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X: &ast.Ident{
																Name: "metric",
															},
															Sel: &ast.Ident{
																Name: "WithAttributes",
															},
														},
														Args: []ast.Expr{
															&ast.CallExpr{
																Fun: &ast.SelectorExpr{
																	X: &ast.Ident{
																		Name: "attribute",
																	},
																	Sel: &ast.Ident{
																		Name: "String",
																	},
																},
																Args: []ast.Expr{
																	&ast.BasicLit{
																		Kind:  token.STRING,
																		Value: "\"query_version\"",
																	},
																	&ast.Ident{
																		Name: setUnexported(name) + "Version",
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})
		}
		if generateErrorMetrics {
			Stmt = append(Stmt, &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.DeferStmt{
						Call: &ast.CallExpr{
							Fun: &ast.FuncLit{
								Type: &ast.FuncType{
									Params: &ast.FieldList{},
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.IfStmt{
											Cond: &ast.BinaryExpr{
												X: &ast.Ident{
													Name: "err",
												},
												Op: token.NEQ,
												Y: &ast.Ident{
													Name: "nil",
												},
											},
											Body: &ast.BlockStmt{
												List: []ast.Stmt{
													&ast.ExprStmt{
														X: &ast.CallExpr{
															Fun: &ast.SelectorExpr{
																X: &ast.SelectorExpr{
																	X: &ast.Ident{
																		Name: "q",
																	},
																	Sel: &ast.Ident{
																		Name: setUnexported(name) + "ErrorCounter",
																	},
																},
																Sel: &ast.Ident{
																	Name: "Add",
																},
															},
															Args: []ast.Expr{
																&ast.Ident{
																	Name: "ctx",
																},
																&ast.BasicLit{
																	Kind:  token.INT,
																	Value: "1",
																},
																&ast.CallExpr{
																	Fun: &ast.SelectorExpr{
																		X: &ast.Ident{
																			Name: "metric",
																		},
																		Sel: &ast.Ident{
																			Name: "WithAttributes",
																		},
																	},
																	Args: []ast.Expr{
																		&ast.CallExpr{
																			Fun: &ast.SelectorExpr{
																				X: &ast.Ident{
																					Name: "attribute",
																				},
																				Sel: &ast.Ident{
																					Name: "String",
																				},
																			},
																			Args: []ast.Expr{
																				&ast.BasicLit{
																					Kind:  token.STRING,
																					Value: "\"query_version\"",
																				},
																				&ast.Ident{
																					Name: setUnexported(name) + "Version",
																				},
																			},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})
		}
		if generateInvocationMetrics {
			Stmt = append(Stmt, &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ExprStmt{
						X: &ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X: &ast.SelectorExpr{
									X: &ast.Ident{
										Name: "q",
									},
									Sel: &ast.Ident{
										Name: setUnexported(name) + "InvocationCounter",
									},
								},
								Sel: &ast.Ident{
									Name: "Add",
								},
							},
							Args: []ast.Expr{
								&ast.Ident{
									Name: "ctx",
								},
								&ast.BasicLit{
									Kind:  token.INT,
									Value: "1",
								},
								&ast.CallExpr{
									Fun: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "metric",
										},
										Sel: &ast.Ident{
											Name: "WithAttributes",
										},
									},
									Args: []ast.Expr{
										&ast.CallExpr{
											Fun: &ast.SelectorExpr{
												X: &ast.Ident{
													Name: "attribute",
												},
												Sel: &ast.Ident{
													Name: "String",
												},
											},
											Args: []ast.Expr{
												&ast.BasicLit{
													Kind:  token.STRING,
													Value: "\"query_version\"",
												},
												&ast.Ident{
													Name: setUnexported(name) + "Version",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			})
		}
		Stmt = append(Stmt, &ast.ReturnStmt{
			Results: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "q",
						},
						Sel: &ast.Ident{
							Name: setUnexported(file.Decls[i].(*ast.FuncDecl).Name.Name),
						},
					},
					Args: func() []ast.Expr {
						var e []ast.Expr
						for _, field := range file.Decls[i].(*ast.FuncDecl).Type.Params.List {
							for _, name := range field.Names {
								e = append(e, &ast.Ident{
									Name: name.Name,
								})
							}

						}
						return e
					}(),
				},
			},
		})

		var f []*ast.Field
		for i2, field := range file.Decls[i].(*ast.FuncDecl).Type.Results.List {
			f = append(f, &ast.Field{
				Doc: file.Decls[i].(*ast.FuncDecl).Type.Results.List[i2].Doc,
				Names: func() []*ast.Ident {
					var g []*ast.Ident

					if field.Names != nil {
						return file.Decls[i].(*ast.FuncDecl).Type.Results.List[i2].Names
					} else {
						g = append(g, &ast.Ident{
							Name: func() string {
								if Ident, ok := field.Type.(*ast.Ident); ok && Ident.Name == "error" {
									return "err"
								} else {
									return "arg" + strconv.Itoa(i2)
								}
							}(),
						})
					}
					return g
				}(),
				Type:    file.Decls[i].(*ast.FuncDecl).Type.Results.List[i2].Type,
				Tag:     file.Decls[i].(*ast.FuncDecl).Type.Results.List[i2].Tag,
				Comment: file.Decls[i].(*ast.FuncDecl).Type.Results.List[i2].Comment,
			})

		}

		t := &ast.FuncDecl{
			Recv: file.Decls[i].(*ast.FuncDecl).Recv,
			Name: &ast.Ident{
				Name: setExported(name),
			},
			Type: &ast.FuncType{
				Params: file.Decls[i].(*ast.FuncDecl).Type.Params,
				Results: &ast.FieldList{
					List: f,
				},
			},
			Body: &ast.BlockStmt{
				List: Stmt,
			},
		}
		file.Decls = append(file.Decls, t)
	}
}

// Generates the version constants, which are build by SHA256-Hashing the sql-query
func generateVersionConstants(decl ast.Decl) (ast.Decl, error) {
	if GenDecl, ok := decl.(*ast.GenDecl); ok && GenDecl.Tok == token.CONST {
		for _, spec := range GenDecl.Specs {
			Sha256 := sha256.New()
			_, err := Sha256.Write([]byte(spec.(*ast.ValueSpec).Values[0].(*ast.BasicLit).Value))
			if err != nil {
				fmt.Println(err)
			}
			encoded := bytes.NewBuffer([]byte{})
			writer := base64.NewEncoder(base64.StdEncoding, encoded)
			_, err = writer.Write(Sha256.Sum(nil))
			if err != nil {
				return nil, err
			}
			err = writer.Close()
			if err != nil {
				return nil, err
			}
			all, err := io.ReadAll(encoded)
			if err != nil {
				return nil, err
			}
			return &ast.GenDecl{
				Tok: token.CONST,
				Specs: []ast.Spec{
					&ast.ValueSpec{
						Names: []*ast.Ident{
							&ast.Ident{
								Name: GenDecl.Specs[0].(*ast.ValueSpec).Names[0].Name + "Version",
							},
						},
						Values: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: "\"" + string(all) + "\"",
							},
						},
					},
				},
			}, nil

		}
	}
	return nil, nil
}
