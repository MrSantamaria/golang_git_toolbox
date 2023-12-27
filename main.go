// main.go
package main

import (
	"fmt"
	"log"

	"github.com/mrsantamaria/pkg/prcreator"
)

func main() {
	filePath := "path/to/your/template.yaml"

	err := prcreator.CreatePRsFromFile(filePath)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pull Requests created successfully!")
}
