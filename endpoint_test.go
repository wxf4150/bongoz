package bongoz

import (
	// "fmt"
	"github.com/maxwellhealth/bongo"
	. "gopkg.in/check.v1"
	// "io/ioutil"
	"encoding/json"
	"labix.org/v2/mgo/bson"
	// "log"
	"net/http"
	"net/http/httptest"
	// "strings"
	// "testing"
)

type Page struct {
	Id         bson.ObjectId `bson:"_id"`
	Content    string
	OtherValue int
}

var config = &bongo.Config{
	ConnectionString: "localhost",
	Database:         "gotest",
}

var connection = bongo.Connect(config)

type Factory struct{}

func (f *Factory) New() interface{} {
	return &Page{}
}

func (s *TestSuite) TearDownTest(c *C) {
	connection.Session.DB(config.Database).DropDatabase()
}

func (s *TestSuite) TearDownSuite(c *C) {
	connection.Session.Close()
}

func (s *TestSuite) TestReadList(c *C) {

	collection := connection.Collection("page")

	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	router := endpoint.getRouter()
	w := httptest.NewRecorder()

	// Add two
	obj1 := &Page{
		Content: "foo",
	}

	obj2 := &Page{
		Content: "bar",
	}

	collection.Save(obj1)
	collection.Save(obj2)

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	response := make([]Page, 2)

	err := json.Unmarshal(w.Body.Bytes(), &response)
	c.Assert(err, Equals, nil)

	c.Assert(response[0].Content, Equals, "foo")
	c.Assert(response[1].Content, Equals, "bar")

}
