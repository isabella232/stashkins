package stashkins

import (
	"bytes"
	"encoding/xml"

	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xoom/jenkins"
)

func templateType(xmlDocument []byte) (jenkins.JobType, error) {
	decoder := xml.NewDecoder(bytes.NewBuffer(xmlDocument))

	var t string
	for {
		token, err := decoder.Token()
		if err != nil {
			return jenkins.Unknown, err
		}
		if v, ok := token.(xml.StartElement); ok {
			t = v.Name.Local
			break
		}
	}

	switch t {
	case "maven2-moduleset":
		return jenkins.Maven, nil
	case "project":
		return jenkins.Freestyle, nil
	}
	return jenkins.Unknown, nil
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

		jobType, err := templateType(data)
		if err != nil {
			log.Printf("stashkins.GetTemplates Skipping template repository record %s: %v\n", file, err)
			continue
		} else {
			if jobType == jenkins.Unknown {
				log.Printf("stashkins.GetTemplates Skipping template repository record (unknown type)  %d: %v\n", file, jobType)
				continue
			}
		}

		t := Template{ProjectKey: projectKey, Slug: slug, JobTemplate: data, JobType: jobType}
		templates = append(templates, t)
	}

	return templates, nil
}
