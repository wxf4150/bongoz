package bongoz

import (
	// "fmt"
	"github.com/justinas/alice"
	"github.com/maxwellhealth/bongo"
	. "gopkg.in/check.v1"
	// "io/ioutil"
	"encoding/json"
	"errors"
	"labix.org/v2/mgo/bson"
	// "log"
	"net/http"
	"net/http/httptest"
	// "net/url"
	// "strings"
	// "time"
	// "testing"
)

type listResponse struct {
	Pagination bongo.PaginationInfo
	Data       []map[string]interface{}
}

func (s *TestSuite) TestReadList(c *C) {

	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

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

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	response := &listResponse{}

	err := json.Unmarshal(w.Body.Bytes(), response)

	c.Assert(err, Equals, nil)

	c.Assert(response.Pagination.Current, Equals, 1)
	c.Assert(response.Pagination.TotalPages, Equals, 1)
	c.Assert(response.Pagination.RecordsOnPage, Equals, 2)
	c.Assert(len(response.Data), Equals, 2)
	// log.Println(response)
}

func (s *TestSuite) TestReadListWithMiddleware(c *C) {
	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	endpoint.Middleware.ReadList = alice.New(errorMiddleware)

	router := endpoint.GetRouter()
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 401)
	c.Assert(w.Body.String(), Equals, "Not Authorized\n")

}

func (s *TestSuite) TestReadListWithFailingPreFindFilter(c *C) {

	filter := func(req *http.Request, q bson.M) (error, int) {
		return errors.New("foo"), 503
	}
	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	endpoint.PreFindFilters.ReadList = []QueryFilter{filter}
	router := endpoint.GetRouter()
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 503)
	c.Assert(w.Body.String(), Equals, "{\"error\":\"foo\"}\n")
}

func (s *TestSuite) TestReadListWithPassingPreFindFilter(c *C) {

	filter := func(req *http.Request, q bson.M) (error, int) {
		q["content"] = "foo"
		return nil, 0
	}
	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	endpoint.PreFindFilters.ReadList = []QueryFilter{filter}
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

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	response := &listResponse{}

	err := json.Unmarshal(w.Body.Bytes(), response)

	c.Assert(err, Equals, nil)

	c.Assert(response.Pagination.Current, Equals, 1)
	c.Assert(response.Pagination.TotalPages, Equals, 1)
	c.Assert(response.Pagination.RecordsOnPage, Equals, 1)
	c.Assert(len(response.Data), Equals, 1)
	// log.Println(response.Data)
	c.Assert(response.Data[0]["Content"], Equals, "foo")
}

func (s *TestSuite) TestReadListWithMultiplePreFindFilters(c *C) {

	filter := func(req *http.Request, q bson.M) (error, int) {
		q["content"] = "foo"
		return nil, 0
	}

	filter2 := func(req *http.Request, q bson.M) (error, int) {
		q["bing"] = "baz"
		return nil, 0
	}
	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	endpoint.PreFindFilters.ReadList = []QueryFilter{filter, filter2}
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

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	response := &listResponse{}

	err := json.Unmarshal(w.Body.Bytes(), response)

	c.Assert(err, Equals, nil)

	c.Assert(response.Pagination.Current, Equals, 0)
	c.Assert(response.Pagination.TotalPages, Equals, 0)
	c.Assert(response.Pagination.RecordsOnPage, Equals, 0)
	c.Assert(len(response.Data), Equals, 0)
	// log.Println(response.Data)
	// c.Assert(response.Data[0]["Content"], Equals, "foo")
}

func (s *TestSuite) TestReadListWithFailingPreResponseFilter(c *C) {

	filter := func(req *http.Request, r *HTTPListResponse) (error, int) {
		return errors.New("bar"), 504
	}
	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	endpoint.PreResponseFilters.ReadList = []ListResponseFilter{filter}
	router := endpoint.GetRouter()
	w := httptest.NewRecorder()

	req, _ := http.NewRequest("GET", "/api/pages", nil)
	router.ServeHTTP(w, req)

	c.Assert(w.Code, Equals, 504)
	c.Assert(w.Body.String(), Equals, "{\"error\":\"bar\"}\n")
}

// Serve a collection of 50 elements
func (s *TestSuite) BenchmarkReadList(c *C) {

	doRequest := func(e *Endpoint) {
		router := e.GetRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/pages", nil)
		router.ServeHTTP(w, req)
	}

	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.Factory = &Factory{}

	for n := 0; n < 50; n++ {
		obj := &Page{
			Content: "foo",
		}
		collection.Save(obj)
	}

	c.ResetTimer()

	for i := 0; i < c.N; i++ {
		doRequest(endpoint)
	}
}
