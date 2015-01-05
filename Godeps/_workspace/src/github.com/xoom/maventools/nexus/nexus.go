package nexus

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/xoom/maventools"
)

type (
	createrepo struct {
		XMLName xml.Name       `xml:"repository"`
		Data    CreateRepoData `xml:"data"`
	}

	CreateRepoData struct {
		XMLName            xml.Name                `xml:"data"`
		Id                 maventools.RepositoryID `xml:"id"`
		Name               string                  `xml:"name"`
		ContentResourceURI string                  `xml:"contentResourceURI"`
		Provider           string                  `xml:"provider"`
		ProviderRole       string                  `xml:"providerRole"`
		Format             string                  `xml:"format"`
		RepoType           string                  `xml:"repoType"`
		RepoPolicy         string                  `xml:"repoPolicy"`
		WritePolicy        string                  `xml:"writePolicy"`
		Exposed            bool                    `xml:"exposed"`
		Browseable         bool                    `xml:"browseable"`
		Indexable          bool                    `xml:"indexable"`
		NotFoundCacheTTL   int                     `xml:"notFoundCacheTTL"`
	}

	// The type retrieved or put to read or mutate a repository group.
	repoGroup struct {
		Data RepositoryGroupData `json:"data"`
	}

	// The payload of a repository group read or mutation.
	RepositoryGroupData struct {
		ID                 maventools.GroupID `json:"id"`
		Provider           string             `json:"provider"`
		Name               string             `json:"name"`
		Repositories       []repository       `json:"repositories"`
		Format             string             `json:"format"`
		RepoType           string             `json:"repoType"`
		Exposed            bool               `json:"exposed"`
		ContentResourceURI string             `json:"contentResourceURI"`
	}

	repository struct {
		Name        string                  `json:"name"`
		ID          maventools.RepositoryID `json:"id"`
		ResourceURI string                  `json:"resourceURI"`
	}

	Client struct {
		maventools.ClientConfig
	}
)

// NewClient creates a new Nexus client implementation on which subsequent service methods are called.  The baseURL typically takes
// the form http://host:port/nexus.  username and password are the credentials of an admin user capable of creating and mutating data
// within Nexus.
func NewClient(baseURL, username, password string) Client {
	return Client{maventools.ClientConfig{BaseURL: baseURL, Username: username, Password: password, HttpClient: &http.Client{}}}
}

// RepositoryExists checks whether a given repository specified by repositoryID exists.
func (client Client) RepositoryExists(repositoryID maventools.RepositoryID) (bool, error) {
	req, err := http.NewRequest("GET", client.BaseURL+"/service/local/repositories/"+string(repositoryID), nil)
	if err != nil {
		return false, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
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

// CreateSnapshotRepository creates a new hosted Maven2 SNAPSHOT repository with the given repositoryID.  The repository name
// will be the same as the repositoryID.  When error is nil, the integer return value is the underlying HTTP response code.
func (client Client) CreateSnapshotRepository(repositoryID maventools.RepositoryID) (int, error) {
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
func (client Client) DeleteRepository(repositoryID maventools.RepositoryID) (int, error) {
	req, err := http.NewRequest("DELETE", client.BaseURL+"/service/local/repositories/"+string(repositoryID), nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		return 0, err
	}

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return resp.StatusCode, fmt.Errorf("Client.DeleteRepository() response: %d\n", resp.StatusCode)
	}

	return resp.StatusCode, nil
}

func (client Client) repositoryGroup(groupID maventools.GroupID) (repoGroup, int, error) {
	req, err := http.NewRequest("GET", client.BaseURL+"/service/local/repo_groups/"+string(groupID), nil)
	if err != nil {
		return repoGroup{}, 0, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Add("Accept", "application/json")

	resp, err := client.HttpClient.Do(req)
	if err != nil {
		return repoGroup{}, 0, err

	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return repoGroup{}, 0, err
	}

	if resp.StatusCode != 200 {
		return repoGroup{}, resp.StatusCode, fmt.Errorf("Client.repositoryGroup() response status: %d (%s)\n", resp.StatusCode, string(data))
	}

	var repogroup repoGroup
	if err := json.Unmarshal(data, &repogroup); err != nil {
		return repoGroup{}, 0, err
	}
	return repogroup, resp.StatusCode, nil
}

// Add RepositoryToGroup adds the given repository specified by repositoryID to the repository group specified by groupID.
func (client Client) AddRepositoryToGroup(repositoryID maventools.RepositoryID, groupID maventools.GroupID) (int, error) {
	repogroup, rc, err := client.repositoryGroup(groupID)
	if err != nil {
		return rc, err
	}

	if rc != 200 {
		log.Printf("Nexus Client.AddRepositoryToGroup() response code: %d\n", rc)
	}

	if repoIsInGroup(repositoryID, repogroup) {
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
func (client Client) RemoveRepositoryFromGroup(repositoryID maventools.RepositoryID, groupID maventools.GroupID) (int, error) {
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

func repoIsInGroup(repositoryID maventools.RepositoryID, group repoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return true
		}
	}
	return false
}

func repoIsNotInGroup(repositoryID maventools.RepositoryID, group repoGroup) bool {
	for _, repo := range group.Data.Repositories {
		if repo.ID == repositoryID {
			return false
		}
	}
	return true
}

func removeRepo(repositoryID maventools.RepositoryID, group *repoGroup) {
	ra := make([]repository, 0)
	for _, repo := range group.Data.Repositories {
		if repo.ID != repositoryID {
			ra = append(ra, repo)
		}
	}
	group.Data.Repositories = ra
}
