package main

import (
    "fmt"
    "github.com/reclaimprotocol/xpath-go/internal/parser"
)

func main() {
    p := parser.NewParser()
    result, err := p.Parse("div")
    if err != nil {
        fmt.Println("Error with 'div':", err)
    } else {
        fmt.Println("Success with 'div':", result)
    }
    
    result, err = p.Parse("//div")
    if err != nil {
        fmt.Println("Error with '//div':", err)
    } else {
        fmt.Println("Success with '//div':", result)
    }
}