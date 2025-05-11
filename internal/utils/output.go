package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

func PrintJSON(data []byte) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, data, "", "  ")
	if err != nil {
		fmt.Println(string(data))
		return
	}
	fmt.Println(prettyJSON.String())
}

func PrintErrorAndExit(msg string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s: %v\n", msg, err)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	}
	os.Exit(1)
}
