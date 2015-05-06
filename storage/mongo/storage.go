package mongo

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/ory-am/event"
	gde "github.com/ory-am/gitdeploy/event"
	"github.com/ory-am/gitdeploy/storage"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

type MongoStorage struct {
	db *mgo.Database
}

const (
	appCollection = "app"
	appEventLogCollection = "appEvents"
)

// NewUserStorage creates a new database session for storing users
func New(database *mgo.Database) *MongoStorage {
	ac := database.C(appCollection)
	ensureIndex(ac, mgo.Index{
		Key:    []string{"id"},
		Unique: true,
	})

	ec := database.C(appEventLogCollection)
	ensureIndex(ec, mgo.Index{
		Key:    []string{"id"},
		Unique: true,
	})

	return &MongoStorage{database}
}

func (s *MongoStorage) Trigger(event string, data interface{}) {
	if e, ok := data.(gde.JobEvent); ok {
		// TODO Ugly...
		e.SetEventName(event)
		if _, err := s.AddDeployEvent(e.GetApp(), e.GetMessage()); err != nil {
			log.Fatal(err.Error())
		}
	}
}

func (s *MongoStorage) AttachAggregate(em *event.EventManager) {
	em.AttachListener("jobs.clone", s)
	em.AttachListener("jobs.deploy", s)
	em.AttachListener("jobs.parse", s)
	em.AttachListener("app.created", s)
	em.AttachListener("app.deployed", s)
	em.AttachListener("jobs.cluster", s)
}

func (s *MongoStorage) AddApp(app string, ttl time.Time, repository string) (*storage.App, error) {
	c := s.db.C(appCollection)
	e := &storage.App{
		ID:         app,
		ExpiresAt:  ttl,
		CreatedAt:  time.Now(),
		Killed:     false,
		Repository: repository,
	}
	return e, c.Insert(e)
}

func (s *MongoStorage) UpdateApp(app *storage.App) error {
	c := s.db.C(appCollection)
	return c.Update(bson.M{"id": app.ID}, app)
}

func (s *MongoStorage) AddDeployEvent(app, message string) (*storage.DeployEvent, error) {
	c := s.db.C(appEventLogCollection)
	e := &storage.DeployEvent{
		ID:        uuid.NewRandom().String(),
		App:       app,
		Message:   message,
		Timestamp: time.Now(),
		Unread:    true,
	}
	return e, c.Insert(e)
}

func (s *MongoStorage) GetApp(id string) (app *storage.App, err error) {
	c := s.db.C(appCollection)
	return app, c.Find(bson.M{"id": id}).One(&app)
}

func (s *MongoStorage) GetAppDeployLogs(app string) (e []*storage.DeployEvent, err error) {
	return e, s.db.C(appEventLogCollection).Find(bson.M{"app": app}).All(&e)
}

func (s *MongoStorage) GetNextUnreadMessage(app string) (*storage.DeployEvent, error) {
	e := new(storage.DeployEvent)
	c := s.db.C(appEventLogCollection)
	err := c.Find(bson.M{
		"app":    app,
		"unread": true,
	}).Sort("+timestamp").One(e)
	return e, err
}

func (s *MongoStorage) GetAppKillList() (apps []*storage.App, err error) {
	c := s.db.C(appCollection)
	err = c.Find(bson.M{
		"expiresat": bson.M{
			"$lt": time.Now(),
		},
		"killed": false,
	}).All(&apps)
	return apps, err
}

func (s *MongoStorage) KillApp(app *storage.App) (err error) {
	c := s.db.C(appCollection)
	app.Killed = true
	return c.Update(bson.M{"id": app.ID}, app)
}

func (s *MongoStorage) DeployEventIsRead(event *storage.DeployEvent) error {
	c := s.db.C(appEventLogCollection)
	event.Unread = false
	return c.Update(bson.M{"id": event.ID}, *event)
}

func ensureIndex(c *mgo.Collection, i mgo.Index) {
	err := c.EnsureIndex(i)
	if err != nil {
		log.Fatalf("Could not ensure index: %s", err)
	}
}
