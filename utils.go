// Copyright 2020 - 2021 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var (
	// matchFirstCap = regexp.MustCompile("([A-Z])([A-Z][a-z])")
	matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// ToSnakeCase converts the provided string to snake_case.
func ToSnakeCase(input string) string {
	// output := matchFirstCap.ReplaceAllString(input, "${1}_${2}")
	output := matchAllCap.ReplaceAllString(input, "${1}_${2}")
	output = strings.ReplaceAll(output, "-", "_")
	return strings.ToLower(output)
}

// GetFileList get a list of file by given path.
func GetFileList(path string) (files []string, err error) {
	var fi os.FileInfo
	fi, err = os.Stat(path)
	if err != nil {
		return
	}
	if fi.IsDir() {
		err = filepath.Walk(path, func(fp string, info os.FileInfo, err error) error {
			files = append(files, fp)
			return nil
		})
		if err != nil {
			return
		}
	}
	files = append(files, path)
	return
}

// PrepareOutputDir provide a method to create the output directory by given
// path.
func PrepareOutputDir(path string) error {
	if path == "" {
		return nil
	}
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}
	return nil
}

// BuildInTypes defines the correspondence between Go, TypeScript, C, Java,
// Rust languages and data types in XSD.
// https://www.w3.org/TR/xmlschema-2/#datatype
var BuildInTypes = map[string][]string{
	"anyType":            {"string", "string", "char", "String", "String", "String"},
	"ENTITIES":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>", "Array"},
	"ENTITY":             {"string", "string", "char", "String", "String", "String"},
	"ID":                 {"string", "string", "char", "String", "String", "String"},
	"IDREF":              {"string", "string", "char", "String", "String", "String"},
	"IDREFS":             {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>", "Array"},
	"NCName":             {"string", "string", "char", "String", "String", "String"},
	"NMTOKEN":            {"string", "string", "char", "String", "String", "String"},
	"NMTOKENS":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>", "Array"},
	"NOTATION":           {"[]string", "Array<string>", "char[]", "List<String>", "Vec<String>", "Array"},
	"Name":               {"string", "string", "char", "String", "String", "String"},
	"QName":              {"xml.Name", "any", "char", "String", "String", "String"},
	"anyURI":             {"string", "string", "char", "QName", "String", "String"},
	"base64Binary":       {"[]byte", "Uint8Array", "char[]", "List<Byte>", "String", "Array"},
	"boolean":            {"bool", "boolean", "bool", "Boolean", "bool", "Boolean"},
	"byte":               {"byte", "any", "char[]", "Byte", "u8", "String"},
	"date":               {"time.Time", "string", "char", "Byte", "u8", "Date"},
	"dateTime":           {"time.Time", "string", "char", "Byte", "u8", "DateTime"},
	"decimal":            {"float64", "number", "float", "Float", "f64", "Float"},
	"double":             {"float64", "number", "float", "Float", "f64", "Float"},
	"duration":           {"string", "string", "char", "String", "String", "String"},
	"float":              {"float", "number", "float", "Float", "f64", "Float"},
	"gDay":               {"time.Time", "string", "char", "String", "String", "String"},
	"gMonth":             {"time.Time", "string", "char", "String", "String", "String"},
	"gMonthDay":          {"time.Time", "string", "char", "String", "String", "String"},
	"gYear":              {"time.Time", "string", "char", "String", "String", "String"},
	"gYearMonth":         {"time.Time", "string", "char", "String", "String", "String"},
	"hexBinary":          {"[]byte", "Uint8Array", "char[]", "List<Byte>", "String", "Array"},
	"int":                {"int", "number", "int", "Integer", "i32", "Integer"},
	"integer":            {"int", "number", "int", "Integer", "i32", "Integer"},
	"language":           {"string", "string", "char", "String", "String", "String"},
	"long":               {"int64", "number", "int", "Long", "i64", "Integer"},
	"negativeInteger":    {"int", "number", "int", "Integer", "i32", "Integer"},
	"nonNegativeInteger": {"int", "number", "int", "Integer", "u32", "Integer"},
	"normalizedString":   {"string", "string", "char", "String", "String", "String"},
	"nonPositiveInteger": {"int", "number", "int", "Integer", "i32", "Integer"},
	"positiveInteger":    {"int", "number", "int", "Integer", "u32", "Integer"},
	"short":              {"int16", "number", "int", "Integer", "i16", "Integer"},
	"string":             {"string", "string", "char", "String", "String", "String"},
	"time":               {"time.Time", "string", "char", "String", "String", "Time"},
	"token":              {"string", "string", "char", "String", "String", "String"},
	"unsignedByte":       {"byte", "any", "char", "Byte", "u8", "String"},
	"unsignedInt":        {"uint32", "number", "unsigned int", "Integer", "u32", "Integer"},
	"unsignedLong":       {"uint64", "number", "unsigned int", "Long", "u64", "Bignum"},
	"unsignedShort":      {"uint16", "number", "unsigned int", "Short", "u16", "Integer"},
	"xml:lang":           {"string", "string", "char", "String", "String", "String"},
	"xml:space":          {"string", "string", "char", "String", "String", "String"},
	"xml:base":           {"string", "string", "char", "String", "String", "String"},
	"xml:id":             {"string", "string", "char", "String", "String", "String"},
}

func getBuildInTypeByLang(value, lang string) (buildType string, ok bool) {
	var supportLang = map[string]int{
		"Go":         0,
		"TypeScript": 1,
		"C":          2,
		"Java":       3,
		"Rust":       4,
		"Ruby":       5,
	}
	var buildInTypes []string
	if buildInTypes, ok = BuildInTypes[value]; !ok {
		return
	}
	buildType = buildInTypes[supportLang[lang]]
	return
}

func getBasefromSimpleType(name string, XSDSchema []interface{}) string {
	for _, ele := range XSDSchema {
		switch v := ele.(type) {
		case *SimpleType:
			if !v.List && !v.Union && v.Name == name {
				return v.Base
			}
		case *Attribute:
			if v.Name == name {
				return v.Type
			}
		case *Element:
			if v.Name == name {
				return v.Type
			}
		}
	}
	return name
}

func getNSPrefix(str string) (ns string) {
	split := strings.Split(str, ":")
	if len(split) == 2 {
		ns = split[0]
		return
	}
	return
}

func trimNSPrefix(str string) (name string) {
	split := strings.Split(str, ":")
	if len(split) == 2 {
		name = split[1]
		return
	}
	name = str
	return
}

// MakeFirstUpperCase make the first letter of a string uppercase.
func MakeFirstUpperCase(s string) string {

	if len(s) < 2 {
		return strings.ToUpper(s)
	}

	bts := []byte(s)

	lc := bytes.ToUpper([]byte{bts[0]})
	rest := bts[1:]

	return string(bytes.Join([][]byte{lc, rest}, nil))
}

// callFuncByName calls the no error or only error return function with
// reflect by given receiver, name and parameters.
func callFuncByName(receiver interface{}, name string, params []reflect.Value) (err error) {
	function := reflect.ValueOf(receiver).MethodByName(name)
	if function.IsValid() {
		rt := function.Call(params)
		if len(rt) == 0 {
			return
		}
		if !rt[0].IsNil() {
			err = rt[0].Interface().(error)
			return
		}
	}
	return
}

// isValidUrl tests a string to determine if it is a well-structured url or
// not.
func isValidURL(toTest string) bool {
	_, err := url.ParseRequestURI(toTest)
	if err != nil {
		return false
	}

	u, err := url.Parse(toTest)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

func fetchSchema(URL string) ([]byte, error) {
	var body []byte
	var client http.Client
	var err error
	resp, err := client.Get(URL)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return body, err
		}
	}
	return body, err
}

func genFieldComment(name, doc, prefix string) string {
	docReplacer := strings.NewReplacer("\n", fmt.Sprintf("\r\n%s ", prefix), "\t", "")
	if doc == "" {
		return fmt.Sprintf("\r\n%s %s ...\r\n", prefix, name)
	}
	return fmt.Sprintf("\r\n%s %s is %s\r\n", prefix, name, docReplacer.Replace(doc))
}
