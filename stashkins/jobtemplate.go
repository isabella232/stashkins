package stashkins

import (
	"bytes"
	"encoding/xml"

	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"fmt"
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

func templateWalker(fileName string, found *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Name() == fileName && info.Mode().IsRegular() {
			*found = append(*found, path)
		}
		return nil
	}
}

func templateKey(project, slug string, projectType jenkins.JobType) string {
	return fmt.Sprintf("%s.%s.%d", project, slug, projectType)
}

func GetTemplates(templateRepositoryURL, branch, cloneIntoDir string) ([]JobTemplate, error) {
	if err := FetchTemplates(templateRepositoryURL, branch, cloneIntoDir); err != nil {
		return nil, err
	}

	continuousTemplateFiles := make([]string, 0)
	if err := filepath.Walk(cloneIntoDir, templateWalker("continuous-template.xml", &continuousTemplateFiles)); err != nil {
		return nil, err
	}

	releaseTemplateFiles := make([]string, 0)
	if err := filepath.Walk(cloneIntoDir, templateWalker("release-template.xml", &releaseTemplateFiles)); err != nil {
		return nil, err
	}

	// A temporary auditing map to track continuous templates.
	continuousTemplateTracker := make(map[string]JobTemplate)

	for _, file := range continuousTemplateFiles {
		truncatedPath := strings.Replace(file, cloneIntoDir+"/", "", 1)
		parts := strings.Split(truncatedPath, "/")
		if len(parts) != 3 {
			Log.Printf("stashkins.GetTemplates Skipping invalid template repository record (Unexpected filesystem layout): %s\n", truncatedPath)
			continue
		}
		projectKey := strings.ToLower(parts[0])
		slug := strings.ToLower(parts[1])

		data, err := ioutil.ReadFile(file)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record (Read template file) %s: %v\n", file, err)
			continue
		}

		jobType, err := templateType(data)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record %s: %v\n", file, err)
			continue
		} else {
			if jobType == jenkins.Unknown {
				Log.Printf("stashkins.GetTemplates Skipping template repository record (unknown type)  %d: %v\n", file, jobType)
				continue
			}
		}

		continuousTemplateTracker[templateKey(projectKey, slug, jobType)] = JobTemplate{ProjectKey: projectKey, Slug: slug, ContinuousJobTemplate: data, JobType: jobType}
	}

	// A temporary auditing map to track release templates.
	releaseTemplateTracker := make(map[string]JobTemplate)

	for _, file := range releaseTemplateFiles {
		truncatedPath := strings.Replace(file, cloneIntoDir+"/", "", 1)
		parts := strings.Split(truncatedPath, "/")
		if len(parts) != 3 {
			Log.Printf("stashkins.GetTemplates Skipping invalid template repository record (Unexpected filesystem layout): %s\n", truncatedPath)
			continue
		}
		projectKey := strings.ToLower(parts[0])
		slug := strings.ToLower(parts[1])

		data, err := ioutil.ReadFile(file)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record (Read template file) %s: %v\n", file, err)
			continue
		}

		jobType, err := templateType(data)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record %s: %v\n", file, err)
			continue
		} else {
			if jobType == jenkins.Unknown {
				Log.Printf("stashkins.GetTemplates Skipping template repository record (unknown type)  %d: %v\n", file, jobType)
				continue
			}
		}

		releaseTemplateTracker[templateKey(projectKey, slug, jobType)] = JobTemplate{ProjectKey: projectKey, Slug: slug, ReleaseJobTemplate: data, JobType: jobType}
	}

	// Augment an existing continuous template with release template data.  Mark this release template as processed by flagging it from the backing map.  This is safe.
	for key, releaseTemplate := range releaseTemplateTracker {
		if continuousTemplate, present := continuousTemplateTracker[key]; present {
			continuousTemplate.ReleaseJobTemplate = releaseTemplate.ReleaseJobTemplate
			delete(releaseTemplateTracker, key)
		}
	}

	templates := make([]JobTemplate, 0)

	// Add continuous templates to result
	for _, v := range continuousTemplateTracker {
		templates = append(templates, v)
	}

	// Add release templates not associated with an existing continuous template
	for _, v := range releaseTemplateTracker {
		templates = append(templates, v)
	}

	return templates, nil
}
