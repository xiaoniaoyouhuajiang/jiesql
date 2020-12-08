package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"jiesql"
)

func main() {
	mb := jiesql.NewMemoryBackend()

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to jiesql.")
	for {
		fmt.Print("# ")
		text, err := reader.ReadString('\n')
		text = strings.Replace(text, "\n", "", -1)

		ast, err := jiesql.Parse(text)
		if err != nil {
			panic(err)
		}

		for _, stmt := range ast.Statements {
			switch stmt.Kind {
			case jiesql.CreateTableKind:
				err = mb.CreateTable(ast.Statements[0].CreateTableStatement)
				if err != nil {
					panic(err)
				}
				fmt.Println("ok")
			case jiesql.InsertKind:
				err = mb.Insert(stmt.InsertStatement)
				if err != nil {
					panic(err)
				}

				fmt.Println("ok")
			case jiesql.SelectKind:
				results, err := mb.Select(stmt.SelectStatement)
				if err != nil {
					panic(err)
				}

				for _, col := range results.Columns {
					fmt.Printf("| %s ", col.Name)
				}
				fmt.Println("|")

				for i := 0; i < 20; i++ {
					fmt.Printf("=")
				}
				fmt.Println()

				for _, result := range results.Rows {
					fmt.Printf("|")

					for i, cell := range result {
						typ := results.Columns[i].Type
						s := ""
						switch typ {
						case jiesql.IntType:
							s = fmt.Sprintf("%d", cell.AsInt())
						case jiesql.TextType:
							s = cell.AsText()
						}

						fmt.Printf(" %s | ", s)
					}

					fmt.Println()
				}

				fmt.Println("ok")
			}
		}
	}
}
