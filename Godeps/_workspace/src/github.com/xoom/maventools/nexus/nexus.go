package nexus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type (
	// The type posted in JSON format to create a new Nexus repository.
	createrepo struct {
		Data CreateRepoData `json:"data"`
	}

	CreateRepoData struct {
		ContentResourceURI string `json:"contentResourceURI"`
		Id                 string `json:"id"`
		Name               string `json:"name"`
		Provider           string `json:"provider"`
		ProviderRole       string `json:"providerRole"`
		Format             string `json:"format"`
		RepoType           string `json:"repoType"`
		RepoPolicy         string `json:"repoPolicy"`
		Exposed            bool   `json:"exposed"`
	}

	// The type retrieved or put to read or mutate a repository group.
	RepoGroup struct {
		Data RepositoryGroupData `json:"data"`
	}

	// The payload of a repository group read or mutation.
	RepositoryGroupData struct {
		ID                 string       `json:"id"`
		Provider           string       `json:"provider"`
		Name               string       `json:"name"`
		Repositories       []repository `json:"repositories"`
		Format             string       `json:"format"`
		RepoType           string       `json:"repoType"`
		Exposed            bool         `json:"exposed"`
		ContentResourceURI string       `json:"contentResourceURI"`
	}

	repository struct {
		Name        string `json:"name"`
		ID          string `json:"id"`
		ResourceURI string `json:resourceURI"`
	}

	// A Nexus client
	Client struct {
		baseURL    string // http://localhost:8081/nexus
		username   string
		password   string
		httpClient *http.Client
	}
)

// NewClient creates a new Nexus client on which subsequent service methods are called.  The baseURL typically takes
// the form http://host:port/nexus.  username and password are the credentials of an admin user capable of creating and mutating data
// within Nexus.
func NewClient(baseURL, username, password string) *Client {
	return &Client{baseURL, username, password, &http.Client{}}
}

// RepositoryExists checks whether a given repository specified by repositoryID exists.
func (client *Client) RepositoryExists(repositoryID string) (bool, error) {
	req, err := http.NewRequest("GET", client.baseURL+"/service/local/repositories/"+repositoryID, nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		return false, err
	}

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return false, fmt.Errorf("Client.RepositoryExists(): unexpected response status: %d\n", resp.StatusCode)
	}

	return resp.StatusCode == 200, nil
}

// CreateRepository creates a new hosted Maven2 SNAPSHOT repository with the given repositoryID.  The repository name
// will be the same as the repositoryID.
func (client *Client) CreateRepository(repositoryID string) error {
	repo := createrepo{
		Data: CreateRepoData{
			Id:                 repositoryID,
			Name:               repositoryID,
			Provider:           "maven2",
			RepoType:           "hosted",
			RepoPolicy:         "SNAPSHOT",
			ProviderRole:       "org.sonatype.nexus.proxy.repository.Repository",
			ContentResourceURI: client.baseURL + "/content/repositories/" + repositoryID,
			Format:             "maven2",
			Exposed:            true,
		}}

	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", client.baseURL+"/service/local/repositories", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		return fmt.Errorf("Client.CreateRepository(): unexpected response status: %d (%s)\n", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRepository deletes the repository with the given repositoryID.
func (client *Client) DeleteRepository(repositoryID string) error {
	req, err := http.NewRequest("DELETE", client.baseURL+"/service/local/repositories/"+repositoryID, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		return err
	}

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return fmt.Errorf("Client.DeleteRepository(): unexpected response status: %d\n", resp.StatusCode)
	}

	return nil
}

// RepositoryGroup gets a repository group specified by groupID.
func (client *Client) RepositoryGroup(groupID string) (RepoGroup, error) {
	req, err := http.NewRequest("GET", client.baseURL+"/service/local/repo_groups/"+groupID, nil)
	if err != nil {
		return RepoGroup{}, err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return RepoGroup{}, err

	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return RepoGroup{}, err
	}

	if resp.StatusCode != 200 {
		return RepoGroup{}, fmt.Errorf("Client.RepositoryGroup(): unexpected response status: %d\n", resp.StatusCode)
	}

	var repogroup RepoGroup
	if err := json.Unmarshal(data, &repogroup); err != nil {
		return RepoGroup{}, err
	}
	return repogroup, nil
}

// Add RepositoryToGroup adds the given repository specified by repositoryID to the repository group specified by groupID.
func (client *Client) AddRepositoryToGroup(repositoryID, groupID string) error {
	repogroup, err := client.RepositoryGroup(groupID)
	if err != nil {
		return err
	}

	if repoIsInGroup(repositoryID, repogroup) {
		return nil
	}

	repo := repository{Name: repositoryID, ID: repositoryID, ResourceURI: client.baseURL + "/service/local/repo_groups/" + groupID + "/" + repositoryID}
	repogroup.Data.Repositories = append(repogroup.Data.Repositories, repo)

	data, err := json.Marshal(&repogroup)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", client.baseURL+"/service/local/repo_groups/"+groupID, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Client.AddRepositoryToGroup(): unexpected response status: %d (%s)\n", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteRepositoryFromGroup removes the given repository specified by repositoryID from the repository group specified by groupID.
func (client *Client) DeleteRepositoryFromGroup(repositoryID, groupID string) error {
	repogroup, err := client.RepositoryGroup(groupID)
	if err != nil {
		return err
	}

	if repoIsNotInGroup(repositoryID, repogroup) {
		return nil
	}

	removeRepo(repositoryID, &repogroup)

	data, err := json.Marshal(&repogroup)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", client.baseURL+"/service/local/repo_groups/"+groupID, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.SetBasicAuth(client.username, client.password)
	req.Header.Add("Content-type", "application/json")
	req.Header.Add("Accept", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Client.AddRepositoryToGroup(): unexpected response status: %d (%s)\n", resp.StatusCode, string(body))
	}

	return nil
}

func repoIsInGroup(repositoryID string, group RepoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return true
		}
	}
	return false
}

func repoIsNotInGroup(repositoryID string, group RepoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return false
		}
	}
	return true
}

func removeRepo(repositoryID string, group *RepoGroup) {
	ra := make([]repository, 0)
	for _, repo := range group.Data.Repositories {
		if repo.ID != repositoryID {
			ra = append(ra, repo)
		}
	}
	group.Data.Repositories = ra
}
