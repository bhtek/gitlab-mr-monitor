package main

import (
	"fmt"
	"log"

	"./adapter"
)

func main() {
	adapter := adapter.GitLabAdapter{ApiKey: "By3ySyMr-9S5yD3zDYsu"}
	projs, err := adapter.SearchProjects()
	if err != nil {
		log.Fatal(err)
	}

	for _, proj := range projs {
		fmt.Println(proj)
	}
}
