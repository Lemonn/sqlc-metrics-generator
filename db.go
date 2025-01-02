package main

import (
	"go/ast"
	"go/token"
	"strings"
)

func modifyDbFile(file *ast.File, foundFunctions []string, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics, generateConnectionRetriever bool) *ast.File {

	if previouslyModified(file) {
		panic("modifyDbFile called more than once")
	}
	addModifiedComment(file)

	requiredImports := []string{
		"context",
		"go.opentelemetry.io/otel/metric",
	}
	addMissingImports(file, requiredImports)

	replaceNewFunction(file, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics)

	generateQueryStruct(file, foundFunctions, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics)
	if generateQueryRuntimeMetrics {
		file.Decls = append(file.Decls, createInitRuntimeMetricsFunction(foundFunctions))
	}
	if generateInvocationMetrics {
		file.Decls = append(file.Decls, createInitCallMetricsFunction(foundFunctions))
	}
	if generateErrorMetrics {
		file.Decls = append(file.Decls, createInitErrorMetricsFunction(foundFunctions))
	}

	if generateConnectionRetriever {
		file.Decls = append(file.Decls, createConnectionRetrievalFunction())
	}

	return file
}

func createConnectionRetrievalFunction() *ast.FuncDecl {
	return &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "q",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: "Queries",
						},
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: "GetConnection",
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{
							Name: "DBTX",
						},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.SelectorExpr{
							X: &ast.Ident{
								Name: "q",
							},
							Sel: &ast.Ident{
								Name: "db",
							},
						},
					},
				},
			},
		},
	}
}

// Replaces the New function, with one that requires a metric meter and a basename
func replaceNewFunction(file *ast.File, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics bool) {

	List := []ast.Stmt{
		&ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X: &ast.Ident{
					Name: "basename",
				},
				Op: token.EQL,
				Y: &ast.Ident{
					Name: "nil",
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.Ident{
								Name: "defaultBasename",
							},
						},
						Tok: token.DEFINE,
						Rhs: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: "\"sqlc\"",
							},
						},
					},
					&ast.AssignStmt{
						Lhs: []ast.Expr{
							&ast.Ident{
								Name: "basename",
							},
						},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{
							&ast.UnaryExpr{
								Op: token.AND,
								X: &ast.Ident{
									Name: "defaultBasename",
								},
							},
						},
					},
				},
			},
		},
		&ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{
					Name: "q",
				},
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.UnaryExpr{
					Op: token.AND,
					X: &ast.CompositeLit{
						Type: &ast.Ident{
							Name: "Queries",
						},
						Elts: []ast.Expr{
							&ast.KeyValueExpr{
								Key: &ast.Ident{
									Name: "db",
								},
								Value: &ast.Ident{
									Name: "db",
								},
							},
							&ast.KeyValueExpr{
								Key: &ast.Ident{
									Name: "meter",
								},
								Value: &ast.Ident{
									Name: "meter",
								},
							},
							&ast.KeyValueExpr{
								Key: &ast.Ident{
									Name: "basename",
								},
								Value: &ast.StarExpr{
									X: &ast.Ident{
										Name: "basename",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if generateQueryRuntimeMetrics {
		List = append(List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "q",
						},
						Sel: &ast.Ident{
							Name: "initRuntimeMetrics",
						},
					},
				},
			},
		})
		List = append(List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "nil",
							},
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}

	if generateInvocationMetrics {
		List = append(List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "q",
						},
						Sel: &ast.Ident{
							Name: "initCallMetrics",
						},
					},
				},
			},
		})
		List = append(List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "nil",
							},
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}

	if generateErrorMetrics {
		List = append(List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.Ident{
							Name: "q",
						},
						Sel: &ast.Ident{
							Name: "initErrorMetrics",
						},
					},
				},
			},
		})
		List = append(List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "nil",
							},
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}
	List = append(List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.Ident{
				Name: "q",
			},
			&ast.Ident{
				Name: "nil",
			},
		},
	})

	for i, decl := range file.Decls {
		if FuncDecl, ok := decl.(*ast.FuncDecl); ok {
			if FuncDecl.Name.Name == "New" {

				file.Decls[i] = &ast.FuncDecl{
					Name: &ast.Ident{
						Name: "New",
					},
					Type: &ast.FuncType{
						Params: &ast.FieldList{
							List: []*ast.Field{
								&ast.Field{
									Names: []*ast.Ident{
										&ast.Ident{
											Name: "db",
										},
									},
									Type: &ast.Ident{
										Name: "DBTX",
									},
								},
								&ast.Field{
									Names: []*ast.Ident{
										&ast.Ident{
											Name: "meter",
										},
									},
									Type: &ast.SelectorExpr{
										X: &ast.Ident{
											Name: "metric",
										},
										Sel: &ast.Ident{
											Name: "Meter",
										},
									},
								},
								&ast.Field{
									Names: []*ast.Ident{
										&ast.Ident{
											Name: "basename",
										},
									},
									Type: &ast.StarExpr{
										X: &ast.Ident{
											Name: "string",
										},
									},
								},
							},
						},
						Results: &ast.FieldList{
							List: []*ast.Field{
								&ast.Field{
									Type: &ast.StarExpr{
										X: &ast.Ident{
											Name: "Queries",
										},
									},
								},
								&ast.Field{
									Type: &ast.Ident{
										Name: "error",
									},
								},
							},
						},
					},
					Body: &ast.BlockStmt{
						List: List,
					},
				}
			}
		}
	}
}

// Add a metric value to the Query struct for each function
func generateQueryStruct(file *ast.File, foundFunctions []string, generateInvocationMetrics, generateErrorMetrics, generateQueryRuntimeMetrics bool) {
	list := []*ast.Field{
		&ast.Field{
			Names: []*ast.Ident{
				&ast.Ident{
					Name: "db",
				},
			},
			Type: &ast.Ident{
				Name: "DBTX",
			},
		},
		&ast.Field{
			Names: []*ast.Ident{
				&ast.Ident{
					Name: "meter",
				},
			},
			Type: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "metric",
				},
				Sel: &ast.Ident{
					Name: "Meter",
				},
			},
		},
		&ast.Field{
			Names: []*ast.Ident{
				&ast.Ident{
					Name: "basename",
				},
			},
			Type: &ast.Ident{
				Name: "string",
			},
		},
	}
	if generateQueryRuntimeMetrics {
		for _, function := range foundFunctions {
			list = append(list, &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: setUnexported(function) + "RuntimeGauge",
					},
				},
				Type: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "metric",
					},
					Sel: &ast.Ident{
						Name: "Float64Gauge",
					},
				},
			})
		}
	}
	if generateInvocationMetrics {
		for _, function := range foundFunctions {
			list = append(list, &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: setUnexported(function) + "InvocationCounter",
					},
				},
				Type: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "metric",
					},
					Sel: &ast.Ident{
						Name: "Int64Counter",
					},
				},
			})
		}
	}
	if generateErrorMetrics {
		for _, function := range foundFunctions {
			list = append(list, &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: setUnexported(function) + "ErrorCounter",
					},
				},
				Type: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "metric",
					},
					Sel: &ast.Ident{
						Name: "Int64Counter",
					},
				},
			})
		}
	}
	for i, decl := range file.Decls {
		//Replace New function
		if GenDecl, ok := decl.(*ast.GenDecl); ok {
			if TypeSpec, ok := GenDecl.Specs[0].(*ast.TypeSpec); ok && TypeSpec.Name.Name == "Queries" {
				file.Decls[i] = &ast.GenDecl{
					Tok: token.TYPE,
					Specs: []ast.Spec{
						&ast.TypeSpec{
							Name: &ast.Ident{
								Name: "Queries",
							},
							Type: &ast.StructType{
								Fields: &ast.FieldList{
									List: list,
								},
							},
						},
					},
				}
			}
		}
	}
}

func createInitRuntimeMetricsFunction(fundFunctions []string) *ast.FuncDecl {
	//Create empty initMetric function
	initMetricsFunction := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "q",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: "Queries",
						},
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: "initRuntimeMetrics",
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},

		Body: &ast.BlockStmt{
			List: []ast.Stmt{},
		},
	}

	//Add error var
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "err",
						},
					},
					Type: &ast.Ident{
						Name: "error",
					},
				},
			},
		},
	})

	//Init metric for each found function
	for _, functionName := range fundFunctions {
		//Init metric
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X: &ast.Ident{
						Name: "q",
					},
					Sel: &ast.Ident{
						Name: setUnexported(functionName) + "RuntimeGauge",
					},
				},
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "q",
							},
							Sel: &ast.Ident{
								Name: "meter",
							},
						},
						Sel: &ast.Ident{
							Name: "Float64Gauge",
						},
					},
					Args: []ast.Expr{
						&ast.ParenExpr{
							X: &ast.BinaryExpr{
								X: &ast.SelectorExpr{
									X: &ast.Ident{
										Name: "q",
									},
									Sel: &ast.Ident{
										Name: "basename",
									},
								},
								Op: token.ADD,
								Y: &ast.BasicLit{
									Kind:  token.STRING,
									Value: "\"" + strings.ToLower(toSnakeCase(functionName)) + "_runtime_gauge\"",
								},
							},
						},
					},
				},
			},
		})
		//If error handler
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.Ident{
				Name: "nil",
			},
		},
	})

	return initMetricsFunction
}

func createInitErrorMetricsFunction(fundFunctions []string) *ast.FuncDecl {
	//Create empty InitErrorMetrics function
	initMetricsFunction := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "q",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: "Queries",
						},
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: "initErrorMetrics",
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},

		Body: &ast.BlockStmt{
			List: []ast.Stmt{},
		},
	}

	//Add error var
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "err",
						},
					},
					Type: &ast.Ident{
						Name: "error",
					},
				},
			},
		},
	})

	//Init metric for each found function
	for _, functionName := range fundFunctions {
		//Init metric
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X: &ast.Ident{
						Name: "q",
					},
					Sel: &ast.Ident{
						Name: setUnexported(functionName) + "ErrorCounter",
					},
				},
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "q",
							},
							Sel: &ast.Ident{
								Name: "meter",
							},
						},
						Sel: &ast.Ident{
							Name: "Int64Counter",
						},
					},
					Args: []ast.Expr{
						&ast.ParenExpr{
							X: &ast.BinaryExpr{
								X: &ast.SelectorExpr{
									X: &ast.Ident{
										Name: "q",
									},
									Sel: &ast.Ident{
										Name: "basename",
									},
								},
								Op: token.ADD,
								Y: &ast.BasicLit{
									Kind:  token.STRING,
									Value: "\"" + strings.ToLower(toSnakeCase(functionName)) + "_error_counter\"",
								},
							},
						},
					},
				},
			},
		})
		//If error handler
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.Ident{
				Name: "nil",
			},
		},
	})

	return initMetricsFunction
}

func createInitCallMetricsFunction(fundFunctions []string) *ast.FuncDecl {
	//Create empty InitErrorMetrics function
	initMetricsFunction := &ast.FuncDecl{
		Recv: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "q",
						},
					},
					Type: &ast.StarExpr{
						X: &ast.Ident{
							Name: "Queries",
						},
					},
				},
			},
		},
		Name: &ast.Ident{
			Name: "initCallMetrics",
		},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{
							Name: "error",
						},
					},
				},
			},
		},

		Body: &ast.BlockStmt{
			List: []ast.Stmt{},
		},
	}

	//Add error var
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.DeclStmt{
		Decl: &ast.GenDecl{
			Tok: token.VAR,
			Specs: []ast.Spec{
				&ast.ValueSpec{
					Names: []*ast.Ident{
						&ast.Ident{
							Name: "err",
						},
					},
					Type: &ast.Ident{
						Name: "error",
					},
				},
			},
		},
	})

	//Init metric for each found function
	for _, functionName := range fundFunctions {
		//Init metric
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X: &ast.Ident{
						Name: "q",
					},
					Sel: &ast.Ident{
						Name: setUnexported(functionName) + "InvocationCounter",
					},
				},
				&ast.Ident{
					Name: "err",
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X: &ast.SelectorExpr{
							X: &ast.Ident{
								Name: "q",
							},
							Sel: &ast.Ident{
								Name: "meter",
							},
						},
						Sel: &ast.Ident{
							Name: "Int64Counter",
						},
					},
					Args: []ast.Expr{
						&ast.ParenExpr{
							X: &ast.BinaryExpr{
								X: &ast.SelectorExpr{
									X: &ast.Ident{
										Name: "q",
									},
									Sel: &ast.Ident{
										Name: "basename",
									},
								},
								Op: token.ADD,
								Y: &ast.BasicLit{
									Kind:  token.STRING,
									Value: "\"" + strings.ToLower(toSnakeCase(functionName)) + "_call_counter\"",
								},
							},
						},
					},
				},
			},
		})
		//If error handler
		initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.IfStmt{
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
					&ast.ReturnStmt{
						Results: []ast.Expr{
							&ast.Ident{
								Name: "err",
							},
						},
					},
				},
			},
		})
	}
	initMetricsFunction.Body.List = append(initMetricsFunction.Body.List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.Ident{
				Name: "nil",
			},
		},
	})

	return initMetricsFunction
}
