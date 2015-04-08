package restserver

import (
	"encoding/json"
	"github.com/justinas/alice"
	"github.com/maxwellhealth/bongo"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"testing"
)

type listResponse struct {
	Pagination bongo.PaginationInfo
	Data       []map[string]interface{}
}

func TestReadList(t *testing.T) {
	conn := getConnection()
	collection := conn.Collection("pages")
	defer conn.Session.Close()

	Convey("ReadList", t, func() {
		endpoint := NewEndpoint("/api/pages", conn, "pages")
		Convey("basic readlist", func() {
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

			err := collection.Save(obj1)
			So(err, ShouldEqual, nil)
			err = collection.Save(obj2)
			So(err, ShouldEqual, nil)

			req, _ := http.NewRequest("GET", "/api/pages", nil)
			router.ServeHTTP(w, req)

			response := &listResponse{}

			err = json.Unmarshal(w.Body.Bytes(), response)

			So(err, ShouldEqual, nil)
			So(response.Pagination.Current, ShouldEqual, 1)
			So(response.Pagination.TotalPages, ShouldEqual, 1)
			So(response.Pagination.RecordsOnPage, ShouldEqual, 2)
			So(len(response.Data), ShouldEqual, 2)
		})
		Convey("with query", func() {
			endpoint.Factory = Factory
			endpoint.AllowFullQuery = true
			router := endpoint.GetRouter()
			w := httptest.NewRecorder()

			// Add two
			obj1 := &Page{
				Content: "foo",
			}

			obj2 := &Page{
				Content: "bar",
			}

			err := collection.Save(obj1)
			So(err, ShouldEqual, nil)
			err = collection.Save(obj2)
			So(err, ShouldEqual, nil)

			req, _ := http.NewRequest("GET", `/api/pages?_query={"content":{"$regex":"OO","$options":"i"}}`, nil)
			router.ServeHTTP(w, req)

			response := &listResponse{}

			err = json.Unmarshal(w.Body.Bytes(), response)

			So(err, ShouldEqual, nil)
			So(response.Pagination.Current, ShouldEqual, 1)
			So(response.Pagination.TotalPages, ShouldEqual, 1)
			So(response.Pagination.RecordsOnPage, ShouldEqual, 1)
			So(len(response.Data), ShouldEqual, 1)
		})
		Convey("readlist with middleware", func() {
			endpoint.Factory = Factory

			endpoint.Middleware.ReadList = alice.New(errorMiddleware)

			router := endpoint.GetRouter()
			w := httptest.NewRecorder()

			req, _ := http.NewRequest("GET", "/api/pages", nil)
			router.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, 401)
			So(w.Body.String(), ShouldEqual, "Not Authorized\n")
		})
		Reset(func() {
			conn.Session.DB("bongoz").DropDatabase()

		})
	})
}

// Serve a collection of 50 elements
func BenchmarkReadList(b *testing.B) {

	conn := getConnection()
	collection := conn.Collection("pages")
	defer func() {
		conn.Session.DB("dpltest").DropDatabase()
		conn.Session.Close()
	}()

	doRequest := func(e *Endpoint) {
		router := e.GetRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/pages", nil)
		router.ServeHTTP(w, req)
	}

	endpoint := NewEndpoint("/api/pages", conn, "pages")
	endpoint.Factory = Factory

	for n := 0; n < 50; n++ {
		obj := &Page{
			Content: "foo",
		}
		collection.Save(obj)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		doRequest(endpoint)
	}
}
