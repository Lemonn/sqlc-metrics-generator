package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
)

func main() {
	var foundFunctions []string
	var err error

	path := flag.String("path", "", "The path to the sqlc output folder")
	queryFilename := flag.String("queryFilename", "query.sql.go", "The name of the query file")
	dbFilename := flag.String("dbFilename", "db.go", "The name of the db file")
	generateInvocationMetrics := flag.Bool("generateInvocationMetrics", false, "Set if invocation metrics should be generated")
	generateErrorMetrics := flag.Bool("generateErrorMetrics", false, "Set if error metrics should be generated")
	generateQueryRuntimeMetrics := flag.Bool("generateQueryRuntimeMetrics", false, "Set if runtime metrics should be generated")
	generateConnectionRetriever := flag.Bool("generateConnectionRetriever", false, "Set to generate a convince function to retrieve the underlying database connection")
	flag.Parse()

	if path == nil {
		s := ""
		path = &s
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, *path+*queryFilename, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}
	if !*generateInvocationMetrics && !*generateErrorMetrics && !*generateQueryRuntimeMetrics {
		fmt.Println("At least one of the metrics needs to be set to true, otherwise this tool does not make sense")
		return
	}
	file, foundFunctions, err = modifyQuerySqlFile(file, *generateInvocationMetrics, *generateErrorMetrics, *generateQueryRuntimeMetrics)
	if err != nil {
		fmt.Println(err)
		return
	}

	output := bytes.NewBuffer([]byte{})
	if err := printer.Fprint(output, fset, file); err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(*path+*queryFilename, output.Bytes(), 0666)
	if err != nil {
		fmt.Println(err)
		return
	}

	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, *path+*dbFilename, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}
	file = modifyDbFile(file, foundFunctions, *generateInvocationMetrics, *generateErrorMetrics, *generateQueryRuntimeMetrics, *generateConnectionRetriever)
	output = bytes.NewBuffer([]byte{})
	if err = printer.Fprint(output, fset, file); err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(*path+*dbFilename, output.Bytes(), 0666)
	if err != nil {
		fmt.Println(err)
		return
	}
}
