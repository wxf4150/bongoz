package bongoz

import (
	// "fmt"
	// "github.com/justinas/alice"
	// "github.com/maxwellhealth/bongo"
	. "gopkg.in/check.v1"
	// "io/ioutil"
	"encoding/json"
	// "errors"
	// "labix.org/v2/mgo/bson"
	// "log"
	"net/http"
	"net/http/httptest"
	// "net/url"
	"strings"
	// "time"
	// "testing"
)

func (s *TestSuite) TestCreate(c *C) {

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = Factory

	router := endpoint.GetRouter()
	w := httptest.NewRecorder()

	obj1 := map[string]string{
		"Content": "foo",
	}

	marshaled, err := json.Marshal(obj1)

	c.Assert(err, Equals, nil)

	reader := strings.NewReader(string(marshaled))
	req, _ := http.NewRequest("POST", "/api/pages", reader)
	router.ServeHTTP(w, req)

	response := &singleResponse{}

	c.Assert(w.Code, Equals, 201)
	err = json.Unmarshal(w.Body.Bytes(), response)

	c.Assert(err, Equals, nil)

	c.Assert(response.Data["content"], Equals, "foo")
}

func (s *TestSuite) TestCreateWithValidationErrors(c *C) {

	endpoint := NewEndpoint("/api/pages", connection, "pages")
	endpoint.Factory = ValidFactory

	router := endpoint.GetRouter()
	w := httptest.NewRecorder()

	obj1 := map[string]string{
		"Content": "",
	}

	marshaled, err := json.Marshal(obj1)

	c.Assert(err, Equals, nil)

	reader := strings.NewReader(string(marshaled))
	req, _ := http.NewRequest("POST", "/api/pages", reader)
	router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 400)
	c.Assert(w.Body.String(), Equals, "{\"error\":[\"Content is required\"]}\n")

}

// func (s *TestSuite) TestReadOneWithPassingPreFindFilter(c *C) {
// 	filter := func(req *http.Request, q bson.M) (error, int) {
// 		q["foo"] = "bar"
// 		return nil, 0
// 	}

// 	endpoint := NewEndpoint("/api/pages", collection)
// 	endpoint.Factory = Factory
// 	endpoint.PreFindFilters.ReadOne = []QueryFilter{filter}

// 	router := endpoint.GetRouter()
// 	w := httptest.NewRecorder()

// 	// Add two
// 	obj1 := &Page{
// 		Content: "foo",
// 	}

// 	obj2 := &Page{
// 		Content: "bar",
// 	}

// 	collection.Save(obj1)
// 	collection.Save(obj2)

// 	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
// 	router.ServeHTTP(w, req)

// 	log.Println(w.Body)
// 	// response := &singleResponse{}

// 	c.Assert(w.Code, Equals, 404)
// }

// func (s *TestSuite) TestReadOneWithFailingPreFindFilter(c *C) {
// 	filter := func(req *http.Request, q bson.M) (error, int) {
// 		return errors.New("test"), 504
// 	}

// 	endpoint := NewEndpoint("/api/pages", collection)
// 	endpoint.Factory = Factory
// 	endpoint.PreFindFilters.ReadOne = []QueryFilter{filter}

// 	router := endpoint.GetRouter()
// 	w := httptest.NewRecorder()

// 	// Add two
// 	obj1 := &Page{
// 		Content: "foo",
// 	}

// 	obj2 := &Page{
// 		Content: "bar",
// 	}

// 	collection.Save(obj1)
// 	collection.Save(obj2)

// 	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
// 	router.ServeHTTP(w, req)

// 	log.Println(w.Body)
// 	// response := &singleResponse{}

// 	c.Assert(w.Code, Equals, 504)

// }

// func (s *TestSuite) TestReadOneWithFailingPreResponseFilter(c *C) {
// 	filter := func(req *http.Request, res *HTTPSingleResponse) (error, int) {
// 		return errors.New("test"), 504
// 	}

// 	endpoint := NewEndpoint("/api/pages", collection)
// 	endpoint.Factory = Factory
// 	endpoint.PreResponseFilters.ReadOne = []SingleResponseFilter{filter}

// 	router := endpoint.GetRouter()
// 	w := httptest.NewRecorder()

// 	// Add two
// 	obj1 := &Page{
// 		Content: "foo",
// 	}

// 	obj2 := &Page{
// 		Content: "bar",
// 	}

// 	collection.Save(obj1)
// 	collection.Save(obj2)

// 	req, _ := http.NewRequest("GET", strings.Join([]string{"/api/pages/", obj1.Id.Hex()}, ""), nil)
// 	router.ServeHTTP(w, req)

// 	log.Println(w.Body)
// 	// response := &singleResponse{}

// 	c.Assert(w.Code, Equals, 504)

// }
