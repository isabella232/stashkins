package maventools

import "net/http"

// Types expected to be common to Nexus and other repo managers
type (

	// ClientOps defines the service methods on a Client.  This interface should be suffiently expressive to capture Nexus and Artifactory behavior.
	// Integer return values are the underlying HTTP response codes.
	IClient interface {
		RepositoryExists(RepositoryID) (bool, error)
		CreateSnapshotRepository(RepositoryID) (int, error)
		DeleteRepository(RepositoryID) (int, error)
		AddRepositoryToGroup(RepositoryID, GroupID) (int, error)
		RemoveRepositoryFromGroup(RepositoryID, GroupID) (int, error)
		RepositoryGroup(GroupID) (RepositoryGroup, int, error)
	}

	RepositoryID string

	GroupID string

	RepositoryGroup struct {
		ID                 GroupID
		Name               string
		ContentResourceURI string
		Repositories       []Repository
	}

	Repository struct {
		Name        string
		ID          RepositoryID
		ResourceURI string
	}

	ClientConfig struct {
		// The public client interface
		IClient
		// For Nexus clients, typically http://host:port/nexus
		BaseURL string
		// Admin username and password capable of updating the artifact repository
		Username string
		Password string
		// Underlying network client
		HttpClient *http.Client
	}
)
