package utils

import (
	"io/ioutil"
	"os"
)

func FileWrite(filename, content string) (err error) {
	f, err := os.Create(filename)
	defer f.Close()
	if err == nil {
		_, err = f.WriteString(content)
	}
	return
}

func FileGet(filename string) (content string, err error) {
	f, err := ioutil.ReadFile(filename)
	if err == nil {
		return string(f), nil
	}
	return "", err
}
