package maventools

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ae6rt/retry"
)

type (
	createrepo struct {
		XMLName xml.Name       `xml:"repository"`
		Data    CreateRepoData `xml:"data"`
	}

	CreateRepoData struct {
		XMLName            xml.Name     `xml:"data"`
		Id                 RepositoryID `xml:"id"`
		Name               string       `xml:"name"`
		ContentResourceURI string       `xml:"contentResourceURI"`
		Provider           string       `xml:"provider"`
		ProviderRole       string       `xml:"providerRole"`
		Format             string       `xml:"format"`
		RepoType           string       `xml:"repoType"`
		RepoPolicy         string       `xml:"repoPolicy"`
		WritePolicy        string       `xml:"writePolicy"`
		Exposed            bool         `xml:"exposed"`
		Browseable         bool         `xml:"browseable"`
		Indexable          bool         `xml:"indexable"`
		NotFoundCacheTTL   int          `xml:"notFoundCacheTTL"`
	}

	// The type retrieved or put to read or mutate a repository group.
	repoGroup struct {
		Data RepositoryGroupData `json:"data"`
	}

	// The payload of a repository group read or mutation.
	RepositoryGroupData struct {
		ID                 GroupID      `json:"id"`
		Provider           string       `json:"provider"`
		Name               string       `json:"name"`
		Repositories       []repository `json:"repositories"`
		Format             string       `json:"format"`
		RepoType           string       `json:"repoType"`
		Exposed            bool         `json:"exposed"`
		ContentResourceURI string       `json:"contentResourceURI"`
	}

	repository struct {
		Name        string       `json:"name"`
		ID          RepositoryID `json:"id"`
		ResourceURI string       `json:"resourceURI"`
	}

	NexusClient struct {
		ClientConfig
	}
)

// NewClient creates a new Nexus client implementation on which subsequent service methods are called.  The baseURL typically takes
// the form http://host:port/nexus.  username and password are the credentials of an admin user capable of creating and mutating data
// within Nexus.
func NewNexusClient(baseURL, username, password string) NexusClient {
	return NexusClient{ClientConfig{BaseURL: baseURL, Username: username, Password: password, HttpClient: &http.Client{}}}
}

// RepositoryExists checks whether a given repository specified by repositoryID exists.
func (client NexusClient) RepositoryExists(repositoryID RepositoryID) (bool, error) {
	retry := retry.New(3*time.Second, 3, retry.DefaultBackoffFunc)
	var exists bool

	work := func() error {
		req, err := http.NewRequest("GET", client.BaseURL+"/service/local/repositories/"+string(repositoryID), nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(client.Username, client.Password)
		req.Header.Add("Accept", "application/json")

		resp, err := client.HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		}

		exists = resp.StatusCode == 200
		if resp.StatusCode != 200 && resp.StatusCode != 404 {
			return fmt.Errorf("Client.RepositoryExists(): unexpected response status: %d\n", resp.StatusCode)
		}
		return nil
	}

	return exists, retry.Try(work)
}

// CreateSnapshotRepository creates a new hosted Maven2 SNAPSHOT repository with the given repositoryID.  The repository name
// will be the same as the repositoryID.  When error is nil, the integer return value is the underlying HTTP response code.
func (client NexusClient) CreateSnapshotRepository(repositoryID RepositoryID) (int, error) {
	repo := createrepo{
		Data: CreateRepoData{
			Id:                 repositoryID,
			Name:               string(repositoryID),
			Provider:           "maven2",
			RepoType:           "hosted",
			RepoPolicy:         "SNAPSHOT",
			ProviderRole:       "org.sonatype.nexus.proxy.repository.Repository",
			ContentResourceURI: client.BaseURL + "/content/repositories/" + string(repositoryID),
			Format:             "maven2",
			Browseable:         true,
			Indexable:          true,
			Exposed:            true,
			WritePolicy:        "ALLOW_WRITE",
			NotFoundCacheTTL:   1440,
		}}

	data, err := xml.Marshal(&repo)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", client.BaseURL+"/service/local/repositories", bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Add("Content-type", "application/xml")
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 201 {
		return resp.StatusCode, fmt.Errorf("Client.CreateSnapshotRepository(): unexpected response status: %d (%s)\n", resp.StatusCode, string(body))
	}

	return resp.StatusCode, nil
}

// DeleteRepository deletes the repository with the given repositoryID.
func (client NexusClient) DeleteRepository(repositoryID RepositoryID) (int, error) {
	retry := retry.New(3*time.Second, 3, retry.DefaultBackoffFunc)
	var responseCode int
	work := func() error {
		req, err := http.NewRequest("DELETE", client.BaseURL+"/service/local/repositories/"+string(repositoryID), nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(client.Username, client.Password)
		req.Header.Add("Accept", "application/json")

		resp, err := client.HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		}

		responseCode = resp.StatusCode
		if responseCode != 204 && responseCode != 404 {
			return fmt.Errorf("Client.DeleteRepository() response: %d\n", responseCode)
		}
		return nil
	}

	return responseCode, retry.Try(work)
}

// RepositoryGroup returns a representation of the given repository group ID.
func (client NexusClient) RepositoryGroup(groupID GroupID) (RepositoryGroup, int, error) {
	repoGroup, rc, err := client.repositoryGroup(groupID)
	if err != nil {
		return RepositoryGroup{}, rc, err
	}
	if rc != 200 {
		return RepositoryGroup{}, rc, nil
	}
	return canonicalize(repoGroup), rc, nil
}

// AddRepositoryToGroup adds the given repository specified by repositoryID to the repository group specified by groupID.
func (client NexusClient) AddRepositoryToGroup(repositoryID RepositoryID, groupID GroupID) (int, error) {
	repogroup, rc, err := client.repositoryGroup(groupID)
	if err != nil {
		return rc, err
	}

	// If there is no error preceding this, rc should always be 200.  But say something if it isn't.
	if rc != 200 {
		log.Printf("Nexus Client.AddRepositoryToGroup() response code: %d\n", rc)
	}

	if repoIsInGroup(repositoryID, repogroup) {
		log.Printf("Nexus Client.AddRepositoryToGroup(): RepositoryID %v is already in repository group.  Will not PUT over HTTP.\n", repositoryID)
		return 0, nil
	}

	repo := repository{ID: repositoryID, Name: string(repositoryID), ResourceURI: client.BaseURL + "/service/local/repo_groups/" + string(groupID) + "/" + string(repositoryID)}
	repogroup.Data.Repositories = append(repogroup.Data.Repositories, repo)

	data, err := json.Marshal(&repogroup)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequest("PUT", client.BaseURL+"/service/local/repo_groups/"+string(groupID), bytes.NewBuffer(data))
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return resp.StatusCode, fmt.Errorf("Client.AddRepositoryToGroup(): unexpected response status: %d (%s)\n", resp.StatusCode, string(body))
	}

	return resp.StatusCode, nil
}

// DeleteRepositoryFromGroup removes the given repository specified by repositoryID from the repository group specified by groupID.
func (client NexusClient) RemoveRepositoryFromGroup(repositoryID RepositoryID, groupID GroupID) (int, error) {
	repogroup, rc, err := client.repositoryGroup(groupID)
	if err != nil {
		return rc, err
	}
	if rc != 200 {
		log.Printf("Nexus Client.AddRepositoryToGroup() response code: %d\n", rc)
	}

	if repoIsNotInGroup(repositoryID, repogroup) {
		return 0, nil
	}

	removeRepo(repositoryID, &repogroup)

	data, err := json.Marshal(&repogroup)
	if err != nil {
		return 0, err
	}

	retry := retry.New(3*time.Second, 3, retry.DefaultBackoffFunc)
	var responseCode int
	work := func() error {
		req, err := http.NewRequest("PUT", client.BaseURL+"/service/local/repo_groups/"+string(groupID), bytes.NewBuffer(data))
		if err != nil {
			return err
		}
		req.SetBasicAuth(client.Username, client.Password)
		req.Header.Add("Content-type", "application/json")
		req.Header.Add("Accept", "application/json")

		resp, err := client.HttpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		responseCode = resp.StatusCode
		if responseCode != 200 {
			return fmt.Errorf("Client.AddRepositoryToGroup(): unexpected response status: %d (%s)\n", responseCode, string(body))
		}
		return nil
	}
	return responseCode, retry.Try(work)
}

func (client NexusClient) repositoryGroup(groupID GroupID) (repoGroup, int, error) {
	retry := retry.New(3*time.Second, 3, retry.DefaultBackoffFunc)
	var data []byte
	var responseCode int
	work := func() error {
		req, err := http.NewRequest("GET", client.BaseURL+"/service/local/repo_groups/"+string(groupID), nil)
		if err != nil {
			return err
		}
		req.SetBasicAuth(client.Username, client.Password)
		req.Header.Add("Accept", "application/json")

		resp, err := client.HttpClient.Do(req)
		if err != nil {
			return err

		}
		defer resp.Body.Close()

		data, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		responseCode = resp.StatusCode
		if responseCode != 200 {
			return fmt.Errorf("Client.repositoryGroup() response status: %d (%s)\n", responseCode, string(data))
		}
		return nil
	}
	if err := retry.Try(work); err != nil {
		return repoGroup{}, responseCode, err
	}

	var repogroup repoGroup
	if err := json.Unmarshal(data, &repogroup); err != nil {
		return repoGroup{}, 0, err
	}
	return repogroup, responseCode, nil
}

func repoIsInGroup(repositoryID RepositoryID, group repoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return true
		}
	}
	return false
}

func repoIsNotInGroup(repositoryID RepositoryID, group repoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return false
		}
	}
	return true
}

func removeRepo(repositoryID RepositoryID, group *repoGroup) {
	ra := make([]repository, 0)
	for _, repo := range group.Data.Repositories {
		if repo.ID != repositoryID {
			ra = append(ra, repo)
		}
	}
	group.Data.Repositories = ra
}

func canonicalize(nexusRepositoryGroup repoGroup) RepositoryGroup {
	c := RepositoryGroup{
		ID:                 nexusRepositoryGroup.Data.ID,
		Name:               nexusRepositoryGroup.Data.Name,
		ContentResourceURI: nexusRepositoryGroup.Data.ContentResourceURI,
		Repositories:       make([]Repository, 0),
	}
	for _, r := range nexusRepositoryGroup.Data.Repositories {
		r := Repository{ID: r.ID, Name: r.Name, ResourceURI: r.ResourceURI}
		c.Repositories = append(c.Repositories, r)
	}
	return c
}
