package bongoz

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/maxwellhealth/bongo"
	"labix.org/v2/mgo/bson"
	// "github.com/oleiade/reflections"
	"io"
	// "labix.org/v2/mgo/bson"
	"errors"
	"log"
	"net/http"
	"time"
	// "net/url"
	// "reflect"
	// "strconv"
	"reflect"
	"strconv"
	"strings"
	// "time"
	"fmt"
	"io/ioutil"
)

type SortConfig struct {
	Field     string
	Direction int
}

type PaginationConfig struct {
	PerPage int
	Sort    []SortConfig
}

type QueryFilter func(*http.Request, bson.M) (error, int)
type DocumentFilter func(*http.Request, interface{}) (error, int)
type ListResponseFilter func(*http.Request, *HTTPListResponse) (error, int)
type SingleResponseFilter func(*http.Request, *HTTPSingleResponse) (error, int)
type PostWriteResponseHook func(*http.Request, string, interface{})
type PostReadResponseHook func(*http.Request, string)

// Use this to inspect the request body, for signature-based security, etc
type PreServeFilter func(*http.Request, []byte) (error, int)

type HTTPListResponse struct {
	Pagination *bongo.PaginationInfo `jsonutils:"pagination"`
	Data       []interface{}         `jsonutils:"data"`
}

type HTTPSingleResponse struct {
	Data interface{} `jsonutils:"data"`
}

type HTTPErrorResponse struct {
	Error error `jsonutils:"error"`
}

func NewErrorResponse(err error) *HTTPErrorResponse {
	return &HTTPErrorResponse{err}
}

func (e *HTTPErrorResponse) ToJSON() string {

	if reflect.TypeOf(e.Error).String() != "*bongo.SaveResult" {
		m := make(map[string]string)
		m["error"] = e.Error.Error()
		marshaled, _ := MarshalJSON(m)
		return string(marshaled)
	} else {
		marshaled, _ := MarshalJSON(e)
		return string(marshaled)
	}

}

type modelFactory interface {
	New() interface{}
}

type PreFindFilters struct {
	ReadOne  []QueryFilter
	ReadList []QueryFilter
	Update   []QueryFilter
	Delete   []QueryFilter
}

type PreSaveFilters struct {
	Create []DocumentFilter
	Update []DocumentFilter
	Delete []DocumentFilter
}

type PostRetrieveFilters struct {
	Update []DocumentFilter
	Delete []DocumentFilter
}

type PreResponseFilters struct {
	ReadOne  []SingleResponseFilter
	ReadList []ListResponseFilter
	Create   []SingleResponseFilter
	Update   []SingleResponseFilter
	Delete   []SingleResponseFilter
}

type Middleware struct {
	ReadOne  alice.Chain
	ReadList alice.Chain
	Create   alice.Chain
	Update   alice.Chain
	Delete   alice.Chain
}

type Endpoint struct {
	Collection             *bongo.Collection
	Uri                    string
	QueryParams            []string
	Pagination             *PaginationConfig
	PreServeFilters        []PreServeFilter
	PreFindFilters         *PreFindFilters
	PreSaveFilters         *PreSaveFilters
	PostRetrieveFilters    *PostRetrieveFilters
	PreResponseFilters     *PreResponseFilters
	PostWriteResponseHooks []PostWriteResponseHook
	PostReadResponseHooks  []PostReadResponseHook
	Factory                modelFactory
	Middleware             *Middleware
	AllowFullQuery         bool
}

func NewEndpoint(uri string, collection *bongo.Collection) *Endpoint {
	endpoint := new(Endpoint)
	endpoint.Uri = uri
	endpoint.Collection = collection
	endpoint.Pagination = &PaginationConfig{}

	endpoint.Middleware = new(Middleware)
	endpoint.PreFindFilters = new(PreFindFilters)
	endpoint.PreSaveFilters = new(PreSaveFilters)
	endpoint.PostRetrieveFilters = new(PostRetrieveFilters)
	endpoint.PreResponseFilters = new(PreResponseFilters)

	return endpoint
}

// func (e *Endpoint) PreFind(method string, filter QueryFilter) *Endpoint {
// 	methods := methodsFromMethod(method)
// 	for _, m := range methods {
// 		e.PreFilterHooks[m] = append(e.PreFilterHooks[m], hook)
// 	}

// 	return e

// }

// func (e *Endpoint) PreSave(method string, hook documentFilter) *Endpoint {
// 	methods := methodsFromMethod(method)
// 	for _, m := range methods {
// 		e.PreSaveHooks[m] = append(e.PreSaveHooks[m], hook)
// 	}
// 	return e
// }

// func (e *Endpoint) PostRetrieve(method string, hook documentFilter) *Endpoint {
// 	methods := methodsFromMethod(method)
// 	for _, m := range methods {
// 		e.PostRetrieveHooks[m] = append(e.PostRetrieveHooks[m], hook)
// 	}

// 	return e
// }

// func (e *Endpoint) PreResponse(method string, hook responseFilter) *Endpoint {
// 	methods := methodsFromMethod(method)
// 	for _, m := range methods {
// 		e.PreResponseHooks[m] = append(e.PreResponseHooks[m], hook)
// 	}
// 	return e
// }

func methodsFromMethod(method string) []string {
	if method == "*" || method == "all" {
		return []string{"ReadOne", "ReadList", "Create", "Update", "Delete"}
	} else if method == "write" {
		return []string{"Create", "Update", "Delete"}
	} else if method == "read" {
		return []string{"ReadOne", "ReadList"}
	} else {
		return []string{method}
	}
}

func (e *Endpoint) SetMiddleware(method string, chain alice.Chain) *Endpoint {
	methods := methodsFromMethod(method)
	for _, m := range methods {
		switch m {
		case "ReadOne":
			e.Middleware.ReadOne = chain
		case "ReadList":
			e.Middleware.ReadList = chain
		case "Create":
			e.Middleware.Create = chain
		case "Update":
			e.Middleware.Update = chain
		case "Delete":
			e.Middleware.Delete = chain
		}
	}
	return e
}

// Get the mux router that can be plugged in as an http handler.
// Gives more flexibility than just using the Register() method which
// registers the router directly on the http root handler.
// Use this is you want to use a subroute, a custom http.Server instance, etc
func (e *Endpoint) GetRouter() *mux.Router {
	r := mux.NewRouter()
	r.Handle(e.Uri, e.Middleware.ReadList.ThenFunc(e.HandleReadList)).Methods("GET")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware.ReadOne.ThenFunc(e.HandleReadOne)).Methods("GET")
	r.Handle(e.Uri, e.Middleware.Create.ThenFunc(e.HandleCreate)).Methods("POST")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware.Update.ThenFunc(e.HandleUpdate)).Methods("PUT")
	r.Handle(strings.Join([]string{e.Uri, "{id}"}, "/"), e.Middleware.Delete.ThenFunc(e.HandleDelete)).Methods("DELETE")
	return r
}

// Register the endpoint to the http root handler. Use GetRouter() for more flexibility
func (e *Endpoint) Register() {
	// Make a new router
	r := e.GetRouter()
	http.Handle("/", r)
}

func handleError(w http.ResponseWriter) {
	var err error
	if r := recover(); r != nil {
		// panic(r)
		// return
		if e, ok := r.(error); ok {
			if e.Error() == "EOF" {
				err = errors.New("Lost database connection unexpectedly")
			} else {
				err = e
			}

		} else if e, ok := r.(string); ok {
			err = errors.New(e)
		} else {
			err = errors.New(fmt.Sprint(r))
		}

		http.Error(w, NewErrorResponse(err).ToJSON(), 500)

	}
}

// Handle a "ReadList" request, including parsing pagination, query string, etc
func (e *Endpoint) HandleReadList(w http.ResponseWriter, req *http.Request) {
	// defer handleError(w)
	w.Header().Set("Content-Type", "application/json")
	var err error
	var code int

	body := []byte{}
	for _, f := range e.PreServeFilters {
		err, code = f(req, body)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	start := time.Now()
	// Get the query
	query, err := e.getQuery(req)

	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusBadRequest)
		return
	}

	// Run pre filters for readList
	for _, f := range e.PreFindFilters.ReadList {
		err, code = f(req, query)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	results := e.Collection.Find(query)

	// Default pagination is 50
	if e.Pagination.PerPage == 0 {
		e.Pagination.PerPage = 50
	}

	perPage := e.Pagination.PerPage
	page := 1

	// Allow override with query vars
	perPageParam := req.URL.Query().Get("_perPage")
	pageParam := req.URL.Query().Get("_page")

	if len(perPageParam) > 0 {
		converted, err := strconv.Atoi(perPageParam)
		// Hard limit to 500 so people can break it
		if err == nil && converted > 0 && converted < 500 {
			perPage = converted
		}
	}

	if len(pageParam) > 0 {
		converted, err := strconv.Atoi(pageParam)

		if err == nil && converted >= 1 {
			page = converted
		}
	}

	pageInfo, err := results.Paginate(perPage, page)

	if err != nil {
		panic(err)
	}

	sortParam := req.URL.Query().Get("_sort")

	if len(sortParam) > 0 {
		sortFields := strings.Split(sortParam, ",")
		results.Query.Sort(sortFields...)
	}
	response := make([]interface{}, 0)

	// res := e.Factory.New()

	for i := 0; i < pageInfo.RecordsOnPage; i++ {
		res := e.Factory.New()
		results.Next(res)

		response = append(response, res)

	}

	httpResponse := &HTTPListResponse{pageInfo, response}

	// Filters can modify the response and optionally return a non-nil error, in which case the server's response will be a new
	// HTTP error with the provided error code. Code defaults to 500 if zero (not set)
	for _, f := range e.PreResponseFilters.ReadList {
		err, code = f(req, httpResponse)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	marshaled, err := MarshalJSON(httpResponse)

	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, string(marshaled))

	elapsed := time.Since(start)
	log.Printf("Request took %s", elapsed)

	// Run post response
	for _, f := range e.PostReadResponseHooks {
		f(req, "readList")
	}
}

func (e *Endpoint) HandleReadOne(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")

	var err error
	var code int
	body := []byte{}

	for _, f := range e.PreServeFilters {
		err, code = f(req, body)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	start := time.Now()
	// Step 1 - make sure provided ID is a valid mongo id hex
	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	query := bson.M{
		"_id": bson.ObjectIdHex(id),
	}

	// Run it through the filters
	for _, f := range e.PreFindFilters.ReadOne {
		err, code = f(req, query)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	// Execute the find
	instance := e.Factory.New()

	// Use a FindOne instead of FindById since the query filters may need
	// to add additional parameters to the search query, aside from just ID.
	// Error here is just if there is no document
	err = e.Collection.FindOne(query, instance)

	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusNotFound)
		return
	}

	httpResponse := &HTTPSingleResponse{instance}

	// Run pre response filters
	for _, f := range e.PreResponseFilters.ReadOne {
		err, code = f(req, httpResponse)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	marshaled, err := MarshalJSON(httpResponse)
	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, string(marshaled))
	elapsed := time.Since(start)
	log.Printf("Request took %s", elapsed)

	// Run post response
	for _, f := range e.PostReadResponseHooks {
		f(req, "readOne")
	}

}

func (e *Endpoint) HandleCreate(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")

	var err error
	var code int
	body, err := ioutil.ReadAll(req.Body)

	for _, f := range e.PreServeFilters {
		err, code = f(req, body)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	start := time.Now()

	// decoder := json.NewDecoder(req.Body)

	obj := e.Factory.New()

	// Instantiate diff tracker
	if trackable, ok := obj.(bongo.Trackable); ok {
		trackable.GetDiffTracker().Reset()
	}

	err = json.Unmarshal(body, obj)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Run pre save filters
	for _, f := range e.PreSaveFilters.Create {
		err, code = f(req, obj)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	result := e.Collection.Save(obj)

	if result.Success == false {
		// Make a new JSON e
		http.Error(w, NewErrorResponse(result).ToJSON(), http.StatusBadRequest)
		return
	}

	httpResponse := &HTTPSingleResponse{obj}

	// Run pre response filters
	for _, f := range e.PreResponseFilters.ReadOne {
		err, code = f(req, httpResponse)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	marshaled, _ := MarshalJSON(httpResponse)

	w.WriteHeader(http.StatusCreated)
	io.WriteString(w, string(marshaled))
	elapsed := time.Since(start)
	log.Printf("Request took %s", elapsed)

	// Run post response
	go func() {
		for _, f := range e.PostWriteResponseHooks {
			f(req, "create", obj)
		}
	}()
}

func (e *Endpoint) HandleUpdate(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")

	var err error
	var code int
	body, err := ioutil.ReadAll(req.Body)

	for _, f := range e.PreServeFilters {
		err, code = f(req, body)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	start := time.Now()

	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		http.Error(w, NewErrorResponse(errors.New("Invalid object ID")).ToJSON(), http.StatusBadRequest)
		return
	}

	query := bson.M{
		"_id": bson.ObjectIdHex(id),
	}

	// Run it through the filters
	for _, f := range e.PreFindFilters.Update {
		err, code = f(req, query)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	// Execute the find
	instance := e.Factory.New()

	// Instantiate diff tracker
	if trackable, ok := instance.(bongo.Trackable); ok {
		trackable.GetDiffTracker().Reset()
	}

	// Use a FindOne instead of FindById since the query filters may need
	// to add additional parameters to the search query, aside from just ID.
	// Error here is just if there is no document
	//
	err = e.Collection.FindOne(query, instance)
	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusNotFound)
		return
	}

	if trackable, ok := instance.(bongo.Trackable); ok {
		trackable.GetDiffTracker().Reset()
	}

	err = json.Unmarshal(body, instance)
	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusBadRequest)
		return
	}

	// Run pre save filters
	for _, f := range e.PreSaveFilters.Update {
		err, code = f(req, instance)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	result := e.Collection.Save(instance)

	if result.Success == false {
		// Make a new JSON e
		http.Error(w, NewErrorResponse(result).ToJSON(), http.StatusBadRequest)
		return
	}

	httpResponse := &HTTPSingleResponse{instance}

	// Run pre response filters
	for _, f := range e.PreResponseFilters.Update {
		err, code = f(req, httpResponse)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	marshaled, _ := MarshalJSON(httpResponse)

	io.WriteString(w, string(marshaled))
	elapsed := time.Since(start)
	log.Printf("Request took %s", elapsed)

	// Run post response
	go func() {
		for _, f := range e.PostWriteResponseHooks {
			f(req, "update", instance)
		}
	}()
}

func (e *Endpoint) HandleDelete(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)

	var err error
	var code int
	body := []byte{}

	for _, f := range e.PreServeFilters {
		err, code = f(req, body)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	start := time.Now()

	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		http.Error(w, NewErrorResponse(errors.New("Invalid object ID")).ToJSON(), http.StatusBadRequest)
		return
	}

	query := bson.M{
		"_id": bson.ObjectIdHex(id),
	}

	// Run it through the filters
	for _, f := range e.PreFindFilters.Update {
		err, code = f(req, query)
		if err != nil {
			break
		}
	}
	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	// Execute the find
	instance := e.Factory.New()

	// Use a FindOne instead of FindById since the query filters may need
	// to add additional parameters to the search query, aside from just ID.
	// Error here is just if there is no document
	//
	err = e.Collection.FindOne(query, instance)
	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusBadRequest)
		return
	}

	// Run pre save filters
	for _, f := range e.PreSaveFilters.Update {
		err, code = f(req, instance)
		if err != nil {
			break
		}
	}

	if err != nil {
		if code <= 0 {
			code = http.StatusInternalServerError
		}
		http.Error(w, NewErrorResponse(err).ToJSON(), code)
		return
	}

	err = e.Collection.Delete(instance)

	if err != nil {
		// Make a new JSON e
		http.Error(w, NewErrorResponse(err).ToJSON(), http.StatusBadRequest)
		return
	}

	io.WriteString(w, "OK")
	elapsed := time.Since(start)
	log.Printf("Request took %s", elapsed)

	// Run post response
	go func() {
		for _, f := range e.PostWriteResponseHooks {
			f(req, "delete", instance)
		}
	}()

}
