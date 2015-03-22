package stashkins

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xoom/jenkins"
)

func templateType(xmlDocument []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewBuffer(xmlDocument))
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}
		if v, ok := token.(xml.StartElement); ok {
			return v.Name.Local, nil
		}
	}
	return "", errors.New("This document has no xml.StartElement")
}

func GetTemplates(templateRepositoryURL, branch, dir string) ([]Template, error) {
	if err := FetchTemplates(templateRepositoryURL, branch, dir); err != nil {
		return nil, err
	}

	templateFiles := make([]string, 0)
	err := filepath.Walk(dir, filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == "template.xml" && info.Mode().IsRegular() {
			templateFiles = append(templateFiles, path)
		}
		return nil
	}))

	if err != nil {
		return nil, err
	}

	templates := make([]Template, 0)
	for _, file := range templateFiles {
		truncatedPath := strings.Replace(file, dir+"/", "", 1)
		parts := strings.Split(truncatedPath, "/")
		if len(parts) != 3 {
			log.Printf("stashkins.GetTemplates Skipping invalid template repository record (Unexpected filesystem layout): %s\n", truncatedPath)
			continue
		}
		projectKey := parts[0]
		slug := parts[1]

		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Printf("stashkins.GetTemplates Skipping template repository record (Read template file) %s: %v\n", file, err)
			continue
		}

		startElement, err := templateType(data)
		if err != nil {
			log.Printf("stashkins.GetTemplates Skipping template repository record (XML parsing error) %s: %v\n", file, err)
			continue
		}

		if startElement != "maven2-moduleset" && startElement != "project" {
			log.Printf("stashkins.GetTemplates Skipping template repository record (unsupported document type) %s: %v\n", startElement)
			continue
		}

		var jobType jenkins.JobType

		switch startElement {
		case "maven2-moduleset":
			jobType = jenkins.Maven
		case "project":
			jobType = jenkins.Freestyle
		}

		t := Template{ProjectKey: projectKey, Slug: slug, JobTemplate: data, JobType: jobType}
		templates = append(templates, t)
	}

	return templates, nil
}
