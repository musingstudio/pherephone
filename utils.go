package main

import (
	"log"
	"io/ioutil"

	"os"
	"encoding/json"
	"strings"
)

//PrettyPrint maps
func PrettyPrint(themap map[string]interface{}){
	b, err := json.MarshalIndent(themap, "", "  ")
	if err != nil {
		log.Println("error:", err)
	}
	log.Print(string(b))
}

func readStringFromFile(filename string) (string, error) {
	fileHandle, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Println("file " + filename + " cannot be opened")
		return "", err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Println("Error reading " + filename + " file")
		return "", err
	}
	return string(byteValue), nil
}

func readJSON(filename string) (map[string]interface{}, error){
	
	fileHandle, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Println("file " + filename + " cannot be opened")
		return nil, err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Println("Error reading " + filename + " file")
		return nil, err
	}
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	return jsonData, nil
}

// func writeJSON(filename string, content string) error{
// 	err := ioutil.WriteFile(filename, []byte(content), 0644)
// 	if err != nil {
// 		log.Printf("Unable to write outbox JSON to file: %+v", err)
// 		return err
// 	}
// 	return nil
// }

func makeURLsaveable(url string) string {
	return strings.Replace(url, "/", "😆", -1)
} 

func bringURLback(mangledURL string) string{
	return strings.Replace(mangledURL, "😆", "/", -1)
}