package bongoz

import (
	"errors"
	"github.com/oleiade/reflections"
	"log"
	"reflect"
	"strings"
)

func getFieldTypeByNameOrBsonTag(name string, obj interface{}) (string, error) {
	field, err := getFieldByNameOrBsonTag(name, obj)

	if err != nil {
		return "", err
	} else {
		return field.Type().String(), nil
	}
}

func getFieldByNameOrBsonTag(name string, obj interface{}) (reflect.Value, error) {
	structTags, _ := reflections.Tags(obj, "bson")

	objValue := reflectValue(obj)
	// field := objValue.FieldByName(prop)

	lname := strings.ToLower(name)

	var lk string
	for k, v := range structTags {
		lk = strings.ToLower(k)

		if lk == lname || v == name {
			field := objValue.FieldByName(k)
			return field, nil
		}
	}

	log.Fatalf("No such field: %s in obj", name)
	return objValue, errors.New("No such field")

}

func propertyIsType(obj interface{}, prop string, t string) bool {
	fieldType, err := getFieldTypeByNameOrBsonTag(prop, obj)

	if err != nil {
		return false
	}

	if t == fieldType {
		return true
	}

	return false
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
