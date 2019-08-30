package main

// Utility functions that don't fit anywhere else

import (
	"github.com/gologme/log"

	"io/ioutil"

	"encoding/json"
	"os"
	"strings"
)

//PrettyPrint maps
func PrettyPrint(themap map[string]interface{}) {
	b, err := json.MarshalIndent(themap, "", "  ")
	if err != nil {
		log.Info("error:", err)
	}
	log.Print(string(b))
}

// readStringFromFile opens a file and reads the contents
// into a string. Initially created to avoid verbosity but
// in fact you'd have to handle the error anyway so it doesn't
// save you much trouble
func readStringFromFile(filename string) (string, error) {
	fileHandle, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Info("file " + filename + " cannot be opened")
		return "", err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading " + filename + " file")
		return "", err
	}
	return string(byteValue), nil
}

// readJSON reads a json file and unmarshalls it into
// a map[string]interface{}
func readJSON(filename string) (map[string]interface{}, error) {
	fileHandle, err := os.Open(filename)
	if os.IsNotExist(err) {
		log.Info("file " + filename + " cannot be opened")
		return nil, err
	}
	byteValue, err := ioutil.ReadAll(fileHandle)
	if err != nil {
		log.Info("Error reading " + filename + " file")
		return nil, err
	}
	jsonData := make(map[string]interface{})
	json.Unmarshal(byteValue, &jsonData)

	return jsonData, nil
}

// makeURLsaveable replaces all the slashes with smiley faces
// so that you can safely use it as a filename. You can quite
// safely return it back without loss (though this is not used
// in pherephone)
func makeURLsaveable(url string) string {
	return strings.Replace(url, "/", "😆", -1)
}

// convert a saveable url (see makeURLsaveable()) back to a URL
func bringURLback(mangledURL string) string {
	return strings.Replace(mangledURL, "😆", "/", -1)
}
