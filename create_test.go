package bongoz

import (
	"encoding/json"
	"github.com/maxwellhealth/bongo"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func getConnection() *bongo.Connection {
	conf := &bongo.Config{
		ConnectionString: "localhost",
		Database:         "bongoz",
	}

	conn, err := bongo.Connect(conf)

	if err != nil {
		panic(err)
	}

	return conn
}

func TestCreate(t *testing.T) {
	conn := getConnection()
	defer conn.Session.Close()

	Convey("POST", t, func() {
		endpoint := NewEndpoint("/api/pages", conn, "pages")

		Convey("Basic create", func() {
			endpoint.Factory = Factory

			router := endpoint.GetRouter()
			w := httptest.NewRecorder()

			reader := strings.NewReader(`{"content":"foo","idValue":null, "_id":"540e05189b2212ee6b1f44d3"}`)
			req, _ := http.NewRequest("POST", "/api/pages", reader)
			router.ServeHTTP(w, req)

			response := &singleResponse{}
			So(w.Code, ShouldEqual, 201)
			err := json.Unmarshal(w.Body.Bytes(), response)

			So(err, ShouldEqual, nil)

			So(response.Data["content"], ShouldEqual, "foo")
			So(response.Data["_id"], ShouldEqual, "540e05189b2212ee6b1f44d3")
		})

		Convey("Create with validation errors", func() {
			endpoint.Factory = ValidFactory

			router := endpoint.GetRouter()
			w := httptest.NewRecorder()

			obj1 := map[string]string{
				"Content": "",
			}

			marshaled, err := json.Marshal(obj1)

			So(err, ShouldEqual, nil)

			reader := strings.NewReader(string(marshaled))
			req, _ := http.NewRequest("POST", "/api/pages", reader)
			router.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, 400)
			So(w.Body.String(), ShouldEqual, "{\"errors\":[\"Content is required\"]}")
		})

		Reset(func() {
			conn.Session.DB("bongoz").DropDatabase()
		})
	})
}
