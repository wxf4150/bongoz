package bongoz

import (
	"encoding/json"
	"github.com/maxwellhealth/mgo/bson"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func addIntToQuery(query bson.M, param string, modifier string, value string) {
	withoutPrefix := strings.TrimPrefix(param, strings.Join([]string{modifier, "_"}, ""))

	parsed, err := strconv.Atoi(value)
	if err == nil {
		sub := bson.M{}
		sub[modifier] = parsed
		query[withoutPrefix] = sub
	}
}

func addDateToQuery(query bson.M, param string, modifier string, value string) {

	withoutPrefix := strings.TrimPrefix(param, strings.Join([]string{modifier, "_"}, ""))

	// Remove date from modifier
	parsed, err := strconv.Atoi(value)
	if err == nil {
		i64 := int64(parsed)
		t := time.Unix(i64, 0)
		sub := bson.M{}
		sub[modifier] = t
		query[withoutPrefix] = sub
	}
}

func addDateOrIntToQuery(instance interface{}, query bson.M, param string, modifier string, value string) {
	withoutPrefix := strings.TrimPrefix(param, strings.Join([]string{modifier, "_"}, ""))

	if propertyIsType(instance, withoutPrefix, "time.Time") {
		addDateToQuery(query, param, modifier, value)
	} else {
		addIntToQuery(query, param, modifier, value)
	}
}

func addValueToQuery(instance interface{}, query bson.M, param string, modifier string, value interface{}) {
	addValueToQueryReplacingModifier(instance, query, param, modifier, strings.Join([]string{modifier, "_"}, ""), value)
}

func addValueToQueryReplacingModifier(instance interface{}, query bson.M, param string, modifier string, replace string, value interface{}) {
	withoutPrefix := strings.TrimPrefix(param, replace)

	sub := bson.M{}

	checkForObjectIdAndAddToQuery(instance, withoutPrefix, modifier, value, sub)

	query[withoutPrefix] = sub
}

func checkForObjectIdAndAddToQuery(instance interface{}, property string, key string, value interface{}, query bson.M) {
	t, err := getFieldTypeByNameOrBsonTag(property, instance)
	if err != nil {
		log.Fatal(err.Error())
		return
	}

	if t == "bson.ObjectId" {
		// Make sure it's valid...
		if val, ok := value.(string); ok {
			if bson.IsObjectIdHex(val) {
				query[key] = bson.ObjectIdHex(val)
			} else {
				log.Fatalf("Invalid object ID %s", val)
				return
			}
		} else {
			log.Println("Could not convert value to string", value)
			return
		}

	} else if t == "[]bson.ObjectId" {
		if val, ok := value.([]string); ok {
			parsed := make([]bson.ObjectId, 0)
			for _, v := range val {
				if bson.IsObjectIdHex(v) {
					parsed = append(parsed, bson.ObjectIdHex(v))
				}
			}
			query[key] = parsed
		} else {
			log.Fatal("Could not parse value as []string", value)
			return
		}
	} else {
		query[key] = value
	}

}

func (e *Endpoint) getQuery(req *http.Request) (bson.M, error) {
	query := req.URL.Query()

	q := bson.M{}

	if e.AllowFullQuery {
		// Marshal the query json into bson.M
		val := query.Get("query")

		if len(val) > 0 {
			err := json.Unmarshal([]byte(val), &q)

			return q, err
		}
	}

	// Get an instance so we can inspect it with reflection
	instance := e.Factory()

	for _, param := range e.QueryParams {
		if val, ok := query[param]; ok {
			if len(val) > 0 {
				if strings.HasPrefix(param, "$lt_") {
					addDateOrIntToQuery(instance, q, param, "$lt", query.Get(param))
				} else if strings.HasPrefix(param, "$gt_") {
					addDateOrIntToQuery(instance, q, param, "$gt", query.Get(param))
				} else if strings.HasPrefix(param, "$gte_") {
					addDateOrIntToQuery(instance, q, param, "$gte", query.Get(param))
				} else if strings.HasPrefix(param, "$lte_") {
					addDateOrIntToQuery(instance, q, param, "$lte", query.Get(param))
				} else if strings.HasPrefix(param, "$in_") {
					addValueToQuery(instance, q, param, "$in", val)
				} else if strings.HasPrefix(param, "$nin_") {
					addValueToQuery(instance, q, param, "$nin_", val)
				} else if strings.HasPrefix(param, "$regex_") {
					addValueToQuery(instance, q, param, "$regex", bson.RegEx{query.Get(param), ""})
				} else if strings.HasPrefix(param, "$regexi_") {
					addValueToQueryReplacingModifier(instance, q, param, "$regex", "$regexi_", bson.RegEx{query.Get(param), "i"})
				} else {
					checkForObjectIdAndAddToQuery(instance, param, param, query.Get(param), q)
				}
			}
		}
	}

	return q, nil
}
