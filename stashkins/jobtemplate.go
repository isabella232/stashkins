package stashkins

import (
	"encoding/xml"
	"errors"
	"strings"
)

func TemplateType(xmlDocument []byte) (string, error) {
	reader := strings.NewReader(string(xmlDocument))
	decoder := xml.NewDecoder(reader)
	for {
		token, err = decoder.Token()
		if err != nil {
			return "", err
		}
		if v, ok := token.(xml.StartElement); ok {
			return v.Name.Local, nil
		}
	}
	return "", errors.New("This document has no xml.StartElement")
}
