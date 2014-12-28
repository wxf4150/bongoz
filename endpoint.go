package bongoz

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/maxwellhealth/bongo"
	"io"
	// "log"
	"labix.org/v2/mgo/bson"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SortConfig struct {
	Field     string
	Direction int
}

type PaginationConfig struct {
	PerPage int
	Sort    []SortConfig
}

type queryFilter interface {
	Run(*http.Request, *url.Values) error
}

type documentFilter interface {
	Run(*http.Request, interface{}) error
}

type responseFilter interface {
	Run(*http.Request, map[string]interface{}) error
}

type modelFactory interface {
	New() interface{}
}

type Endpoint struct {
	Collection        *bongo.Collection
	Uri               string
	QueryParams       []string
	Pagination        *PaginationConfig
	PreFilterHooks    map[string][]queryFilter
	PreSaveHooks      map[string][]documentFilter
	PostRetrieveHooks map[string][]documentFilter
	PreResponseHooks  map[string][]responseFilter
	Factory           modelFactory
	Middleware        map[string]alice.Chain
}

func NewEndpoint(uri string, collection *bongo.Collection) *Endpoint {
	endpoint := new(Endpoint)
	endpoint.Uri = uri
	endpoint.Collection = collection
	endpoint.Pagination = &PaginationConfig{}
	methods := []string{"create", "update", "readOne", "readList", "delete"}

	endpoint.PreFilterHooks = make(map[string][]queryFilter)
	endpoint.PreSaveHooks = make(map[string][]documentFilter)
	endpoint.PostRetrieveHooks = make(map[string][]documentFilter)
	endpoint.PreResponseHooks = make(map[string][]responseFilter)
	endpoint.Middleware = make(map[string]alice.Chain)
	for _, m := range methods {
		endpoint.PreFilterHooks[m] = make([]queryFilter, 0)
		endpoint.PreSaveHooks[m] = make([]documentFilter, 0)
		endpoint.PostRetrieveHooks[m] = make([]documentFilter, 0)
		endpoint.PreResponseHooks[m] = make([]responseFilter, 0)
		endpoint.Middleware[m] = alice.Chain{}
	}

	return endpoint
}

func methodsFromMethod(method string) []string {
	var methods []string
	if method == "*" || method == "all" {
		methods = []string{"create", "update", "readOne", "readList", "delete"}
	} else if method == "write" {
		methods = []string{"create", "update", "delete"}
	} else if method == "read" {
		methods = []string{"readOne", "readList"}
	} else {
		methods = []string{method}
	}
	return methods
}

func (e *Endpoint) PreFilter(method string, hook queryFilter) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		e.PreFilterHooks[m] = append(e.PreFilterHooks[m], hook)
	}

	return e

}

func (e *Endpoint) PreSave(method string, hook documentFilter) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		e.PreSaveHooks[m] = append(e.PreSaveHooks[m], hook)
	}
	return e
}

func (e *Endpoint) PostRetrieve(method string, hook documentFilter) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		e.PostRetrieveHooks[m] = append(e.PostRetrieveHooks[m], hook)
	}

	return e
}

func (e *Endpoint) PreResponse(method string, hook responseFilter) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		e.PreResponseHooks[m] = append(e.PreResponseHooks[m], hook)
	}
	return e
}

func (e *Endpoint) SetMiddleware(method string, chain alice.Chain) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		e.Middleware[m] = chain
	}
	return e
}

func (e *Endpoint) getRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle(e.Uri, e.Middleware["readList"].ThenFunc(e.HandleReadList)).Methods("GET")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware["readOne"].ThenFunc(e.HandleReadOne)).Methods("GET")
	r.Handle(e.Uri, e.Middleware["create"].ThenFunc(e.HandleCreate)).Methods("POST")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware["update"].ThenFunc(e.HandleUpdate)).Methods("PUT")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware["delete"].ThenFunc(e.HandleUpdate)).Methods("DELETE")
	return r
}

func (e *Endpoint) Register() {
	// Make a new router
	r := e.getRouter()
	http.Handle("/", r)
}

func addIntToQuery(query *bson.M, modifier string, value string) {
	withoutPrefix := strings.TrimPrefix(param, strings.join([]string{modifier, "_"}, ""))

	parsed, err := strconv.Atoi(value)
	if err == nil {
		sub := &bson.M{}
		sub[modifier] = parsed
		ret[withoutPrefix] = sub
	}
}

func addDateToQuery(query *bson.M, modifier string, value string) {

	withoutPrefix := strings.TrimPrefix(param, strings.join([]string{modifier, "_"}, ""))

	// Remove date from modifier
	parsed, err := strconv.Atoi(value)
	if err == nil {
		t := time.Unix(parsed)
		sub := &bson.M{}
		sub[modifier] = t
		ret[withoutPrefix] = sub
	}
}

func propertyIsType(obj interface{}, prop string, t string) bool {
	objValue := reflectValue(obj)
	field := objValue.FieldByName(prop)
	if !field.IsValid() {
		log.Fatal("No such field: %s in obj", name)
		return false
	}

	name := field.Type().Name()

	if name == t {
		return true
	}

	return false
}

func addDateOrIntToQuery(instance interface{}, query *bson.M, modifier string, value string) {
	withoutPrefix := strings.TrimPrefix(param, "$lt_")
	if propertyIsType(instance, withoutPrefix, "Time") {
		addDateToQuery(query, modifier, value)
	} else {
		addIntToQuery(query, modifier, value)
	}
}

func reflectValue(obj interface{}) reflect.Value {
	var val reflect.Value

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		val = reflect.ValueOf(obj).Elem()
	} else {
		val = reflect.ValueOf(obj)
	}

	return val
}

func (e *Endpoint) getQuery(req *http.Request) *bson.M {
	ret := &bson.M{}
	query := req.URL.Query()

	// Get an instance so we can inspect it with reflection
	instance := e.Factory.New()

	var withoutPrefix string

	for _, param := range e.QueryParams {
		val := query.Get(param)

		if len(val) > 0 {

			if strings.HasPrefix(param, "$lt_") {
				addDateOrIntToQuery(instance, query, "$lt", val)
			} else if strings.HasPrefix(param, "$gt_") {
				addDateOrIntToQuery(instance, query, "$gt", val)
			} else if strings.HasPrefix(param, "$gte_") {
				addDateOrIntToQuery(instance, query, "$gte", val)
			} else if strings.HasPrefix(param, "$lte_") {
				addDateOrIntToQuery(instance, query, "$lte", val)
			} else if strings.HasPrefix(param, "$ltdate_") {
				addDateOrIntToQuery(instance, query, "$ltdate", val)
			} else if strings.HasPrefix(param, "$gtdate_") {
				addDateOrIntToQuery(instance, query, "$gtdate", val)
			} else if strings.HasPrefix(param, "$gtedate_") {
				addDateOrIntToQuery(instance, query, "$gtedate", val)
			} else if strings.HasPrefix(param, "$ltedate_") {
				addDateOrIntToQuery(instance, query, "$ltedate", val)
			}
		}
	}

	return query
}

func (e *Endpoint) HandleReadList(w http.ResponseWriter, req *http.Request) {
	results := e.Collection.Find(nil)

	// Default pagination is 50
	if e.Pagination.PerPage == 0 {
		e.Pagination.PerPage = 50
	}
	pageInfo, err := results.Paginate(e.Pagination.PerPage, 1)

	if err != nil {
		panic(err)
	}
	response := make([]interface{}, 0)

	// res := e.Factory.New()

	for i := 0; i < pageInfo.RecordsOnPage; i++ {
		res := e.Factory.New()
		results.Next(res)

		response = append(response, res)

	}

	marshaled, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	io.WriteString(w, string(marshaled))
}

func (e *Endpoint) HandleReadOne(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func (e *Endpoint) HandleCreate(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func (e *Endpoint) HandleUpdate(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func (e *Endpoint) HandleDelete(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}
