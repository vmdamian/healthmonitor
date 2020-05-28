package gateways

import (
	"context"
	"github.com/gocql/gocql"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	keyspace = "healthmonitor"

	usersTable = "users"
	tokensTable = "tokens"

	userPasswordSelectQuery = "SELECT password FROM " + usersTable + " WHERE id = ?"
	userPasswordInsertQuery = "INSERT INTO " + usersTable + " (id, password) VALUES (?, ?)"

	userByTokenSelectQuery = "SELECT id FROM " + tokensTable + " WHERE user_token = ? ALLOW FILTERING"

	userTokenSelectQuery = "SELECT user_token FROM " + tokensTable + " WHERE id = ?"
	userTokenInsertQuery = "INSERT INTO " + tokensTable + " (id, user_token) VALUES(?, ?)"
	userTokenDeleteQuery = "DELETE FROM " + tokensTable + " WHERE id = ? IF EXISTS"
)

type UsersRepo struct {
	Cluster *gocql.ClusterConfig
	Session *gocql.Session
}

func NewUsersRepo(host string) *UsersRepo {
	cluster := gocql.NewCluster(host)
	cluster.Keyspace = keyspace
	cluster.ConnectTimeout = time.Second * 10

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

func (ur *UsersRepo) RegisterUser(ctx context.Context, username string, cryptedPassword string) error {
	err := ur.Session.Query(userPasswordInsertQuery, username, cryptedPassword).WithContext(ctx).Exec()
	return err
}

func (ur *UsersRepo) LoginUser(ctx context.Context, username string, cryptedPassword string) (bool, string, error) {
	var receivedCryptedPassword string

	// Check if username and password are valid.
	err := ur.Session.Query(userPasswordSelectQuery, username).Consistency(gocql.One).WithContext(ctx).Scan(&receivedCryptedPassword)
	if err != nil {
		log.Infoln(username + receivedCryptedPassword)
		return false, "", err
	}

	if cryptedPassword != receivedCryptedPassword {
		return false, "", nil
	}

	// If username and password are valid generate an authentication token and invalidate a previous one if it exists.
	token, err := ur.getUserToken(ctx, username)
	if err != nil && err != gocql.ErrNotFound {
		return false, "", err
	}

	if token != "" {
		err = ur.deleteUserToken(ctx, username)
		if err != nil {
			return false, "", err
		}
	}

	newToken := uuid.New().String()
	err = ur.insertUserToken(ctx, username, newToken)
	if err != nil {
		return false, "", err
	}

	return true, newToken, nil
}

//TODO: Make this accept a token instead of username and password.
func (ur *UsersRepo) AuthToken(ctx context.Context, token string) (bool, error) {
	var receivedUsername string

	err := ur.Session.Query(userByTokenSelectQuery, token).Consistency(gocql.One).WithContext(ctx).Scan(&receivedUsername)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (ur *UsersRepo) getUserToken(ctx context.Context, username string) (string, error) {
	var receivedToken string

	err := ur.Session.Query(userTokenSelectQuery, username).Consistency(gocql.One).WithContext(ctx).Scan(&receivedToken)
	if err != nil {
		return "", err
	}

	return receivedToken, nil
}

func (ur *UsersRepo) insertUserToken(ctx context.Context, username string, token string) error {
	err := ur.Session.Query(userTokenInsertQuery, username, token).WithContext(ctx).Exec()
	return err
}

func (ur *UsersRepo) deleteUserToken(ctx context.Context, username string) error {
	err := ur.Session.Query(userTokenDeleteQuery, username).Consistency(gocql.One).WithContext(ctx).Exec()
	return err
}

