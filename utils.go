package main

import (
	"encoding/json"
	"fmt"
)

//PrettyPrint maps
func PrettyPrint(themap map[string]interface{}){
	b, err := json.MarshalIndent(themap, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Print(string(b))
}