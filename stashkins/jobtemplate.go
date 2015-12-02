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

func jobType(xmlDocument []byte) (jenkins.JobType, error) {
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

// templateWalker returns a filepath.Walker that finds files named fileName.  Found files are returned in the input string array.
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

// templateKey returns a key that indexes a map entry holding a JobTemplate pointer.
func templateKey(project, slug string, projectType jenkins.JobType) string {
	return fmt.Sprintf("%s.%s.%d", project, slug, projectType)
}

// projectCoordinates returns ("proj", "slug", nil) For input "/prefix/proj/slug/c.xml".  If the input is too short
// to encompass proj/slug/c.xml an error is returned.  IF the input does not end in .xml an error is returned.
func projectCoordinates(fullPath string) (string, string, error) {
	s := strings.ToLower(fullPath)
	if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	parts := strings.Split(s, "/")

	if len(parts) < 3 {
		return "", "", fmt.Errorf("stashkins.GetTemplates Skipping invalid template repository record (Unexpected filesystem layout): %s\n", fullPath)
	}
	if !strings.HasSuffix(parts[len(parts)-1], ".xml") {
		return "", "", fmt.Errorf("stashkins.GetTemplates Skipping invalid template file not ending in .xml: %s\n", fullPath)
	}

	return parts[len(parts)-3], parts[len(parts)-2], nil
}

// buildTemplates iterates over the input file set and builds a map of pointers to JobTemplates.  We need pointer to a JobTemplate so we can mutate it
// when augmenting found continuous templates with associated release template data.
func buildTemplates(files []string, f func(projectKey, slug string, data []byte, jobType jenkins.JobType) *JobTemplate) map[string]*JobTemplate {
	templates := make(map[string]*JobTemplate)

	for _, file := range files {
		projectKey, slug, err := projectCoordinates(file)
		if err != nil {
			Log.Printf("%v\n", err)
			continue
		}

		data, err := ioutil.ReadFile(file)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record (Read template file) %s: %v\n", file, err)
			continue
		}

		jobType, err := jobType(data)
		if err != nil {
			Log.Printf("stashkins.GetTemplates Skipping template repository record %s: %v\n", file, err)
			continue
		} else {
			if jobType == jenkins.Unknown {
				Log.Printf("stashkins.GetTemplates Skipping template repository record (unknown type)  %s: %v\n", file, jobType)
				continue
			}
		}

		templates[templateKey(projectKey, slug, jobType)] = f(projectKey, slug, data, jobType)
	}
	return templates
}

func Templates(templateRepositoryURL, branch, cloneIntoDir string) ([]JobTemplate, error) {
	if err := cloneTemplates(templateRepositoryURL, branch, cloneIntoDir); err != nil {
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
	continuousTemplates := buildTemplates(continuousTemplateFiles, func(projectKey, slug string, data []byte, jobType jenkins.JobType) *JobTemplate {
		return &JobTemplate{ProjectKey: projectKey, Slug: slug, ContinuousJobTemplate: data, JobType: jobType}
	})

	// A temporary auditing map to track release templates.
	releaseTemplates := buildTemplates(releaseTemplateFiles, func(projectKey, slug string, data []byte, jobType jenkins.JobType) *JobTemplate {
		return &JobTemplate{ProjectKey: projectKey, Slug: slug, ReleaseJobTemplate: data, JobType: jobType}
	})

	// Augment existing continuous templates with release templates.  Mark this release template as processed by flagging it in the backing map --- this is safe.
	for key, releaseTemplate := range releaseTemplates {
		if continuousTemplate, present := continuousTemplates[key]; present {
			continuousTemplate.ReleaseJobTemplate = releaseTemplate.ReleaseJobTemplate
			delete(releaseTemplates, key)
		}
	}

	templates := make([]JobTemplate, 0)

	// Add continuous templates to result
	for _, template := range continuousTemplates {
		templates = append(templates, *template)
	}

	// Add release templates not associated with an existing continuous template.  This would be an odd, but possible, condition.
	for _, template := range releaseTemplates {
		templates = append(templates, *template)
	}

	return templates, nil
}
