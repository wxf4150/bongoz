package bongoz

import (
	"encoding/json"
	. "gopkg.in/check.v1"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

func testQuery(testCase *queryTestCase, c *C) {
	var combined string
	if len(testCase.queryString) > 0 {
		combined = strings.Join([]string{"http://localhost:8000?", testCase.queryString}, "")
	} else {
		combined = strings.Join([]string{"http://localhost:8000?", testCase.param, "=", testCase.value}, "")
	}
	parsed, _ := url.Parse(combined)

	log.Println("Checking URL: ", parsed)
	request := &http.Request{
		URL: parsed,
	}

	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.QueryParams = []string{testCase.param}
	endpoint.Factory = Factory
	query, _ := endpoint.getQuery(request)

	c.Assert(query, JSONEquals, testCase.match)

	if testCase.matchType {
		t1 := reflect.TypeOf(query[testCase.param])
		t2 := reflect.TypeOf(testCase.matchTypeTo)

		c.Assert(t1, Equals, t2)
	}
}

type queryTestCase struct {
	queryString string
	param       string
	value       string
	match       bson.M
	matchType   bool
	matchTypeTo interface{}
}

func (s *TestSuite) TestQueryGeneration(c *C) {

	objId := bson.NewObjectId()

	// log.Println(objId.String())
	objId2 := bson.NewObjectId()

	cases := []*queryTestCase{
		&queryTestCase{"", "$regexi_content", "f", bson.M{"content": bson.M{"$regex": bson.RegEx{"f", "i"}}}, false, nil},
		&queryTestCase{"", "$regex_content", "f", bson.M{"content": bson.M{"$regex": bson.RegEx{"f", ""}}}, false, nil},
		&queryTestCase{"", "$gte_intvalue", "5", bson.M{"intvalue": bson.M{"$gte": 5}}, false, nil},
		&queryTestCase{"", "idvalue", objId.Hex(), bson.M{"idvalue": objId}, true, objId},
		&queryTestCase{strings.Join([]string{"$in_idarr=", objId.Hex(), "&$in_idarr=", objId2.Hex()}, ""), "$in_idarr", "", bson.M{"idarr": bson.M{"$in": []bson.ObjectId{objId, objId2}}}, false, nil},
		&queryTestCase{"", "$lt_datevalue", "12345", bson.M{"datevalue": bson.M{"$lt": time.Unix(12345, 0)}}, false, nil},
	}

	for _, testCase := range cases {
		testQuery(testCase, c)
	}
}

func (s *TestSuite) TestFullQuery(c *C) {
	parsed, _ := url.Parse(`http://localhost:8000?query={"_id":{"$oid":12345}}`)

	log.Println("Checking URL: ", parsed)
	request := &http.Request{
		URL: parsed,
	}

	endpoint := NewEndpoint("/api/pages", collection)
	endpoint.AllowFullQuery = true
	query, _ := endpoint.getQuery(request)

	marshaled, _ := json.Marshal(query)
	c.Assert(string(marshaled), Equals, `{"_id":{"$oid":12345}}`)
}
