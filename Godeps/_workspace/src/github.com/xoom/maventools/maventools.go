package maventools

import "net/http"

type RepositoryID string

type GroupID string

type ClientConfig struct {
	// The public client interface
	Client
	// For Nexus clients, typically http://host:port/nexus
	BaseURL string
	// Admin username and password capable of updating the artifact repository
	Username string
	Password string
	// Underlying network client
	HttpClient *http.Client
}

// ClientOps defines the service methods on a Client. Integer return values are the underlying HTTP response codes.
type Client interface {
	RepositoryExists(RepositoryID) (bool, error)
	CreateSnapshotRepository(RepositoryID) (int, error)
	DeleteRepository(RepositoryID) (int, error)
	AddRepositoryToGroup(RepositoryID, GroupID) (int, error)
	RemoveRepositoryFromGroup(RepositoryID, GroupID) (int, error)
}
