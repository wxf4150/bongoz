package restserver

import (
	// "encoding/base64"
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/url"
	"testing"
	// "time"
)

func TestQuery(t *testing.T) {
	conn := getConnection()
	defer conn.Session.Close()

	Convey("Query parsing", t, func() {
		parsed, _ := url.Parse(`http://localhost:8000?_query={"_id":{"$oid":"5525444a91692844dbfef192"}}`)
		// parsed, _ := url.Parse(`http://localhost:8000?_query=HgAAAANkYXRlABMAAAAJJGd0ZQDb7bVmSwEAAAAA`)

		request := &http.Request{
			URL: parsed,
		}

		endpoint := NewEndpoint("/api/pages", conn, "pages")
		endpoint.AllowFullQuery = true
		query, _ := endpoint.getQuery(request)

		// log.Println(query)
		marshaled, _ := json.Marshal(query)
		So(string(marshaled), ShouldEqual, `{"_id":{"$oid":"5525444a91692844dbfef192"}}`)
	})
}
