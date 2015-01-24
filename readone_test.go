package bongoz

import (
	// "fmt"
	// "github.com/justinas/alice"
	// "github.com/maxwellhealth/bongo"
	. "gopkg.in/check.v1"
	// "io/ioutil"
	"encoding/json"
	"errors"
	"github.com/maxwellhealth/mgo/bson"
	"log"
	"net/http"
	"net/http/httptest"
	// "net/url"
	"strings"
	// "time"
	// "testing"
)

func (s *TestSuite) TestReadOne(c *C) {

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = Factory

	router := endpoint.GetRouter()
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

	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
	router.ServeHTTP(w, req)

	log.Println(w.Body)
	response := &singleResponse{}

	c.Assert(w.Code, Equals, 200)
	err := json.Unmarshal(w.Body.Bytes(), response)

	c.Assert(err, Equals, nil)

	c.Assert(response.Data["content"], Equals, "foo")
}

func (s *TestSuite) TestReadOneWithPassingPreFindFilter(c *C) {
	filter := func(req *http.Request, method string, q bson.M) (error, int) {
		q["foo"] = "bar"
		return nil, 0
	}

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = Factory
	endpoint.PreFindFilters = []QueryFilter{filter}

	router := endpoint.GetRouter()
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

	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
	router.ServeHTTP(w, req)

	log.Println(w.Body)
	// response := &singleResponse{}

	c.Assert(w.Code, Equals, 404)
}

func (s *TestSuite) TestReadOneWithFailingPreFindFilter(c *C) {
	filter := func(req *http.Request, method string, q bson.M) (error, int) {
		return errors.New("test"), 504
	}

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = Factory
	endpoint.PreFindFilters = []QueryFilter{filter}

	router := endpoint.GetRouter()
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

	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
	router.ServeHTTP(w, req)

	log.Println(w.Body)
	// response := &singleResponse{}

	c.Assert(w.Code, Equals, 504)

}

func (s *TestSuite) TestReadOneWithFailingPreResponseFilter(c *C) {
	filter := func(req *http.Request, method string, res *HTTPSingleResponse) (error, int) {
		return errors.New("test"), 504
	}

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = Factory
	endpoint.PreResponseSingleFilters = []SingleResponseFilter{filter}

	router := endpoint.GetRouter()
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

	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
	router.ServeHTTP(w, req)

	log.Println(w.Body)
	// response := &singleResponse{}

	c.Assert(w.Code, Equals, 504)

}
