package bongoz

import (
	"github.com/maxwellhealth/bongo"
	. "gopkg.in/check.v1"
	"labix.org/v2/mgo/bson"
	"log"
	"net/http"
	"time"
	// "net/url"
	"encoding/json"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type TestSuite struct{}

var _ = Suite(&TestSuite{})

type NullWriter int

func (NullWriter) Write([]byte) (int, error) { return 0, nil }

func (s *TestSuite) SetUpTest(c *C) {

	if !testing.Verbose() {
		log.SetOutput(new(NullWriter))
	}

}

func errorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Authorized", http.StatusUnauthorized)
		// h.ServeHTTP(w, r)
	})
}

type singleResponse struct {
	Data map[string]interface{}
}

type Page struct {
	Id        bson.ObjectId `bson:"_id" json:"_id"`
	Content   string
	IntValue  int
	DateValue time.Time
	ArrValue  []string
	IdArr     []bson.ObjectId
	IdValue   bson.ObjectId `json:",omitempty" bson:"idValue,omitempty"`
}

var config = &bongo.Config{
	ConnectionString: "localhost",
	Database:         "gotest",
}

var connection, _ = bongo.Connect(config)
var collection = connection.Collection("page")

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

type jsonEqualsChecker struct {
	*CheckerInfo
}

func (checker *jsonEqualsChecker) Check(params []interface{}, names []string) (result bool, error string) {
	json1, err := json.Marshal(params[0])
	if err != nil {
		return false, err.Error()
	}
	json2, err := json.Marshal(params[1])
	if err != nil {
		return false, err.Error()
	}

	return string(json1) == string(json2), ""
}

var JSONEquals Checker = &jsonEqualsChecker{
	&CheckerInfo{Name: "JSONEquals", Params: []string{"obtained", "expected"}},
}

type validatedModel struct {
	Id      bson.ObjectId `bson:"_id",json:"_id"`
	Content string        `json:"content"`
}

type validFactory struct{}

func (f *validFactory) New() interface{} {
	return &validatedModel{}
}

func (v *validatedModel) Validate() []string {
	ret := []string{}
	if !bongo.ValidateRequired(v.Content) {
		ret = append(ret, "Content is required")
	}

	return ret
}
