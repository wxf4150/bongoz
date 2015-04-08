package restserver

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/maxwellhealth/bongo"
	"github.com/maxwellhealth/go-enhanced-json"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
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

type HTTPListResponse struct {
	Pagination *bongo.PaginationInfo
	Data       []interface{}
}

type HTTPSingleResponse struct {
	Data interface{}
}

type HTTPErrorResponse struct {
	Errors []error
}

func NewErrorResponse(err error) *HTTPErrorResponse {

	return &HTTPErrorResponse{[]error{err}}
}

func (e *HTTPErrorResponse) ToJSON() string {

	errs := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		errs[i] = err.Error()
	}
	mp := map[string]interface{}{
		"errors": errs,
	}

	marshaled, _ := json.Marshal(mp)
	return string(marshaled)

}

type HTTPMultiErrorResponse struct {
	Errors []string
}

func NewMultiErrorResponse(errs []error) *HTTPMultiErrorResponse {
	// This is only from json unmarshal
	parsed := make([]string, len(errs))

	for i, e := range errs {
		parsed[i] = e.Error()
	}
	return &HTTPMultiErrorResponse{parsed}
}

func (e *HTTPMultiErrorResponse) ToJSON() string {
	marshaled, _ := json.Marshal(e)
	return string(marshaled)
}

type ModelFactory func() bongo.Document

type Middleware struct {
	ReadOne  alice.Chain
	ReadList alice.Chain
	Create   alice.Chain
	Update   alice.Chain
	Delete   alice.Chain
}

type Endpoint struct {
	CollectionName string
	Connection     *bongo.Connection
	Uri            string
	QueryParams    []string
	Pagination     *PaginationConfig
	Factory        ModelFactory
	Middleware     *Middleware

	AllowFullQuery bool
	DisableWrites  bool
}

func NewEndpoint(uri string, connection *bongo.Connection, collectionName string) *Endpoint {
	endpoint := new(Endpoint)
	endpoint.Uri = uri
	endpoint.Connection = connection
	endpoint.CollectionName = collectionName
	endpoint.Pagination = &PaginationConfig{}
	endpoint.Middleware = new(Middleware)
	return endpoint
}

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
	e.registerRoutes(r)
	return r
}

func (e *Endpoint) registerRoutes(r *mux.Router) {
	r.Handle(e.Uri, e.Middleware.ReadList.ThenFunc(e.HandleReadList)).Methods("GET")
	r.Handle(e.Uri+"/{id}", e.Middleware.ReadOne.ThenFunc(e.HandleReadOne)).Methods("GET")

	if !e.DisableWrites {
		r.Handle(e.Uri, e.Middleware.Create.ThenFunc(e.HandleCreate)).Methods("POST")

		r.Handle(e.Uri+"/{id}", e.Middleware.Update.ThenFunc(e.HandleUpdate)).Methods("PUT")
		r.Handle(e.Uri+"/{id}", e.Middleware.Delete.ThenFunc(e.HandleDelete)).Methods("DELETE")
	}

}

// Register the endpoint to the http root handler. Use GetRouter() for more flexibility
func (e *Endpoint) Register(r *mux.Router) {
	e.registerRoutes(r)
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
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")
	var err error
	var code int

	// Get the query
	query, err := e.getQuery(req)

	if err != nil {
		w.WriteHeader(code)
		io.WriteString(w, NewErrorResponse(err).ToJSON())

		return
	}

	connection := e.Connection

	results := connection.Collection(e.CollectionName).Find(query)

	defer results.Free()

	// Default pagination is 50
	if e.Pagination.PerPage == 0 {
		e.Pagination.PerPage = 50
	}

	perPage := e.Pagination.PerPage
	limit := 0
	skip := 0
	page := 1

	// Allow override with query vars
	perPageParam := req.URL.Query().Get("_perPage")
	pageParam := req.URL.Query().Get("_page")

	// Allow support for limit and skip with no pagination
	limitParam := req.URL.Query().Get("_limit")
	skipParam := req.URL.Query().Get("_skip")

	paginate := true

	if len(limitParam) > 0 {
		paginate = false
		converted, err := strconv.Atoi(limitParam)

		if err == nil && converted > 0 {
			limit = converted
		}
	}

	if len(skipParam) > 0 {

		converted, err := strconv.Atoi(skipParam)
		paginate = false
		if err == nil && converted >= 0 {
			skip = converted
		}
	}

	var pageInfo *bongo.PaginationInfo

	if limit > 0 {
		results.Query.Limit(limit).Skip(skip)
		pageInfo = &bongo.PaginationInfo{}
		pageInfo.TotalPages = 1
		pageInfo.Current = 1
	}

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

	var total int
	if paginate {
		pageInfo, err = results.Paginate(perPage, page)
		if err != nil {
			panic(err)
		}

		total = pageInfo.RecordsOnPage

	} else {
		total, err = results.Query.Count()
		if err != nil {
			panic(err)
		}

		pageInfo.RecordsOnPage = total
		pageInfo.TotalRecords = total
		pageInfo.PerPage = total
		if total == 0 {
			pageInfo.TotalPages = 0
		}
	}

	sortParam := req.URL.Query().Get("_sort")
	if len(sortParam) > 0 {
		sortFields := strings.Split(sortParam, ",")
		results.Query.Sort(sortFields...)
	}

	response := make([]interface{}, total)
	for i := 0; i < total; i++ {
		res := e.Factory()
		results.Next(res)
		response[i] = res

	}

	httpResponse := &HTTPListResponse{pageInfo, response}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(httpResponse)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
	}
}

func (e *Endpoint) HandleReadOne(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")

	var err error

	// Step 1 - make sure provided ID is a valid mongo id hex
	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		http.Error(w, "Invalid object ID", http.StatusBadRequest)
		return
	}

	// Execute the find
	instance := e.Factory()

	err = e.Connection.Collection(e.CollectionName).FindById(bson.ObjectIdHex(id), instance)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
		return
	}

	httpResponse := &HTTPSingleResponse{instance}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(httpResponse)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
	}
}

func (e *Endpoint) HandleCreate(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)

	w.Header().Set("Content-Type", "application/json")

	var err error

	// start := time.Now()

	decoder := json.NewDecoder(req.Body)

	obj := e.Factory()

	// Instantiate diff tracker
	if trackable, ok := obj.(bongo.Trackable); ok {
		trackable.GetDiffTracker().Reset()
	}

	err = decoder.Decode(obj)

	if err != nil {
		if merr, ok := err.(*json.MultipleUnmarshalTypeError); ok {

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, NewMultiErrorResponse(merr.Errors).ToJSON())
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, NewErrorResponse(err).ToJSON())
			return
		}

	}

	err = e.Connection.Collection(e.CollectionName).Save(obj)

	if err != nil {
		if verr, ok := err.(*bongo.ValidationError); ok {
			w.WriteHeader(http.StatusBadRequest)
			errResponse := &HTTPErrorResponse{verr.Errors}
			io.WriteString(w, errResponse.ToJSON())
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, NewErrorResponse(err).ToJSON())
		}
		return
	}

	httpResponse := &HTTPSingleResponse{obj}

	encoder := json.NewEncoder(w)
	w.WriteHeader(http.StatusCreated)
	err = encoder.Encode(httpResponse)

	if err != nil {
		panic(err)
	}

}

func (e *Endpoint) HandleUpdate(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)
	w.Header().Set("Content-Type", "application/json")

	var err error

	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, NewErrorResponse(errors.New("Invalid Object ID")).ToJSON())
		return
	}

	// Execute the find
	instance := e.Factory()

	err = e.Connection.Collection(e.CollectionName).FindById(bson.ObjectIdHex(id), instance)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
		return
	}

	if trackable, ok := instance.(bongo.Trackable); ok {
		trackable.GetDiffTracker().Reset()
	}

	// Save the ID and reapply it afterward, so we do not allow the http request to modify the ID
	actualId := instance.GetId()

	decoder := json.NewDecoder(req.Body)
	err = decoder.Decode(instance)

	if err != nil {
		if merr, ok := err.(*json.MultipleUnmarshalTypeError); ok {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, NewMultiErrorResponse(merr.Errors).ToJSON())
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, NewErrorResponse(err).ToJSON())
			return
		}

	}

	instance.SetId(actualId)

	if tt, ok := instance.(bongo.TimeTracker); ok {
		tt.SetModified(time.Now())
	}

	err = e.Connection.Collection(e.CollectionName).Save(instance)

	if err != nil {
		if verr, ok := err.(*bongo.ValidationError); ok {
			w.WriteHeader(http.StatusBadRequest)
			errResponse := &HTTPErrorResponse{verr.Errors}
			io.WriteString(w, errResponse.ToJSON())
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, NewErrorResponse(err).ToJSON())
		}
		return
	}

	httpResponse := &HTTPSingleResponse{instance}

	encoder := json.NewEncoder(w)
	err = encoder.Encode(httpResponse)

	if err != nil {
		panic(err)
	}

}

func (e *Endpoint) HandleDelete(w http.ResponseWriter, req *http.Request) {
	defer handleError(w)

	var err error

	vars := mux.Vars(req)

	id := vars["id"]

	if len(id) == 0 || !bson.IsObjectIdHex(id) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, NewErrorResponse(errors.New("Invalid Object ID")).ToJSON())
		return
	}

	// Execute the find
	instance := e.Factory()

	// Use a FindOne instead of FindById since the query filters may need
	// to add additional parameters to the search query, aside from just ID.
	// Error here is just if there is no document
	collection := e.Connection.Collection(e.CollectionName)

	err = collection.FindById(bson.ObjectIdHex(id), instance)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
		return
	}

	err = collection.DeleteDocument(instance)

	if err != nil {
		// Make a new JSON e
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, NewErrorResponse(err).ToJSON())
		return
	}

}
