package restserver

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadOne(t *testing.T) {
	conn := getConnection()
	collection := conn.Collection("pages")
	defer conn.Session.Close()

	Convey("ReadOne", t, func() {
		endpoint := NewEndpoint("/api/pages", conn, "pages")
		Convey("basic", func() {
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

			response := &singleResponse{}

			So(w.Code, ShouldEqual, 200)
			err := json.Unmarshal(w.Body.Bytes(), response)

			So(err, ShouldEqual, nil)

			So(response.Data["content"], ShouldEqual, "foo")
		})
		Reset(func() {
			conn.Session.DB("dplservertest").DropDatabase()

		})
	})
}
