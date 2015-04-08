package restserver

import (
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDelete(t *testing.T) {
	conn := getConnection()
	collection := conn.Collection("pages")
	defer conn.Session.Close()

	Convey("DELETE", t, func() {
		endpoint := NewEndpoint("/api/pages", conn, "pages")
		Convey("Basic delete", func() {
			endpoint.Factory = Factory

			router := endpoint.GetRouter()
			w := httptest.NewRecorder()

			obj := &Page{
				Content:  "Foo",
				IntValue: 5,
			}

			err := collection.Save(obj)
			So(err, ShouldEqual, nil)

			req, _ := http.NewRequest("DELETE", strings.Join([]string{"/api/pages", obj.Id.Hex()}, "/"), nil)
			router.ServeHTTP(w, req)

			So(w.Code, ShouldEqual, 200)
			pagination, _ := collection.Find(nil).Paginate(50, 1)

			So(pagination.TotalRecords, ShouldEqual, 0)
		})

		Reset(func() {
			conn.Session.DB("bongoz").DropDatabase()

		})
	})
}
