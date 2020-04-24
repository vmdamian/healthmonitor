package gateways

import (
	"github.com/gocql/gocql"
)

const (
	keyspace = "healthmonitor"
	table = "users"

	userPasswordSelectQuery = "SELECT password FROM " + table + " WHERE id = ?"
	userPasswordInsertQuery = "INSERT INTO " + table + " (id, password) VALUES (?, ?)"
)

type UsersRepo struct {
	Cluster *gocql.ClusterConfig
	Session *gocql.Session
}

func NewUsersRepo(host string) *UsersRepo {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace

	return &UsersRepo{
		Cluster: cluster,
	}
}

func (ur *UsersRepo) Start() error {
	session, err := ur.Cluster.CreateSession()
	if err != nil {
		return err
	}

	ur.Session = session
	return nil
}

func (ur *UsersRepo) RegisterUser(username string, cryptedPassword string) error {
	err := ur.Session.Query(userPasswordInsertQuery, username, cryptedPassword).Exec()
	return err
}

func (ur *UsersRepo) LoginUser(username string, cryptedPassword string) (bool, string, error) {
	var receivedCryptedPassword string

	err := ur.Session.Query(userPasswordSelectQuery, username).Consistency(gocql.One).Scan(&receivedCryptedPassword)
	if err != nil {
		return false, "", err
	}

	if cryptedPassword != receivedCryptedPassword {
		return false, "", nil
	}

	//TODO: Generate token and store it (eventually with expiration time).
	return true, "authenticationToken", nil
}

//TODO: Make this accept a token instead of username and password.
func (ur *UsersRepo) AuthUser(username string, cryptedPassword string) (bool, error) {
	var receivedCryptedPassword string

	err := ur.Session.Query(userPasswordSelectQuery, username).Consistency(gocql.One).Scan(&receivedCryptedPassword)
	if err != nil {
		return false, err
	}

	if cryptedPassword != receivedCryptedPassword {
		return false, nil
	}

	return true, nil
}
