// Copyright 2020 - 2021 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

// CodeGenerator holds code generator overrides and runtime data that are used
// when generate code from proto tree.

var rubyBuildinType = map[string]bool{
	"Array":    true,
	"Decimal":  true,
	"decimal":  true,
	"Float":    true,
	"float":    true,
	"String":   true,
	"string":   true,
	"Time":     true,
	"Date":     true,
	"DateTime": true,
	"Integer":  true,
	"Number":   true,
}

// genRuby generate Ruby programming language source code for XML schema
// definition files.
func (gen *CodeGenerator) GenRuby() error {
	for _, ele := range gen.ProtoTree {
		if ele == nil {
			continue
		}
		funcName := fmt.Sprintf("Ruby%s", reflect.TypeOf(ele).String()[6:])
		callFuncByName(gen, funcName, []reflect.Value{reflect.ValueOf(ele)})
	}
	f, err := os.Create(gen.File + ".rb")
	if err != nil {
		return err
	}
	defer f.Close()
	source := []byte(fmt.Sprintf("# frozen_string_literal: true\n\n%s\n\nrequire 'xmlmapper'\n\nmodule Ota\n\t%s\nend", `# Code generated by xgen. DO NOT EDIT.`, gen.Field))
	f.Write(source)
	return err
}

func genRubyFieldName(name string) (fieldName string) {
	for _, str := range strings.Split(name, ":") {
		fieldName += MakeFirstUpperCase(str)
	}
	var tmp string
	for _, str := range strings.Split(fieldName, ".") {
		tmp += MakeFirstUpperCase(str)
	}
	fieldName = tmp
	fieldName = strings.Replace(strings.Replace(fieldName, "-", "", -1), "_", "", -1)
	return
}

func genRubyFieldType(name string) string {
	if _, ok := rubyBuildinType[name]; ok {
		return name
	}
	var fieldType string
	for _, str := range strings.Split(name, ".") {
		fieldType += MakeFirstUpperCase(str)
	}
	fieldType = strings.Replace(MakeFirstUpperCase(strings.Replace(fieldType, "-", "", -1)), "_", "", -1)
	if fieldType != "" {
		return fieldType
	}
	return "String"
}

// RubySimpleType generates code for simple type XML schema in Ruby language
// syntax.
func (gen *CodeGenerator) RubySimpleType(v *SimpleType) {
	if v.List {
		if _, ok := gen.StructAST[v.Name]; !ok {
			fieldType := genRubyFieldType(getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree))
			content := fmt.Sprintf(" %s", genRubyFieldType(fieldType))
			gen.StructAST[v.Name] = content
			fieldName := genRubyFieldName(v.Name)
			gen.Field += fmt.Sprintf("%s\nclass %s < %s; end\n", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
			return
		}
	}
	if v.Union && len(v.MemberTypes) > 0 {
		if _, ok := gen.StructAST[v.Name]; !ok {
			content := ""
			fieldName := genRubyFieldName(v.Name)
			if fieldName != v.Name {
				gen.ImportEncodingXML = true
				content += fmt.Sprintf("\t\ttag \"%s\"\n", v.Name)
			}

			for memberName, memberType := range v.MemberTypes {
				if memberType == "" { // fix order issue
					memberType = getBasefromSimpleType(memberName, gen.ProtoTree)
				}
				// content += fmt.Sprintf("\t%s\t%s\n", ToSnakeCase(genRubyFieldName(memberName)), genRubyFieldType(memberType))
				content += fmt.Sprintf("\t\tattribute :%s, 'OTA::%s', tag: '%s'\n", ToSnakeCase(genRubyFieldName(memberName)), genRubyFieldType(memberType), memberName)
			}
			content += "\tend\n"
			gen.StructAST[v.Name] = content
			gen.Field += fmt.Sprintf("\t%s\tclass %s\n\t\tinclude XmlMapper\n\n%s\n", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
		}
		return
	}
	if _, ok := gen.StructAST[v.Name]; !ok {
		content := fmt.Sprintf(" %s", genRubyFieldType(getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)))
		gen.StructAST[v.Name] = content
		fieldName := genRubyFieldName(v.Name)
		gen.Field += fmt.Sprintf("\t%s\tclass %s <%s; end\n", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}

// RubyComplexType generates code for complex type XML schema in Ruby language
// syntax.
func (gen *CodeGenerator) RubyComplexType(v *ComplexType) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		content := ""
		fieldName := genRubyFieldName(v.Name)
		if fieldName != v.Name {
			gen.ImportEncodingXML = true
			content += fmt.Sprintf("\t\ttag \"%s\"\n", v.Name)
		}
		for _, attrGroup := range v.AttributeGroup {
			// fmt.Printf("%s\n", getBasefromSimpleType(trimNSPrefix(attrGroup.Ref), gen.ProtoTree))
			fieldType := getBasefromSimpleType(trimNSPrefix(attrGroup.Ref), gen.ProtoTree)
			content += fmt.Sprintf("\t\telement :%s, 'OTA::%s', tag: '%s'\n", ToSnakeCase(genRubyFieldName(attrGroup.Name)), genRubyFieldType(fieldType), genRubyFieldName(attrGroup.Name))
		}

		for _, attribute := range v.Attributes {
			var plural string = "attribute"
			if attribute.Plural {
				plural = "has_many"
			}
			fieldType := genRubyFieldType(getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree))
			content += fmt.Sprintf("\t\t%s :%s, 'OTA::%s', tag: '%s'\n", plural, ToSnakeCase(genRubyFieldName(attribute.Name)), fieldType, attribute.Name)
		}
		for _, group := range v.Groups {
			var plural string
			if group.Plural {
				plural = ""
			}
			content += fmt.Sprintf("\t%s\t%s%s\n", ToSnakeCase(genRubyFieldName(group.Name)), plural, genRubyFieldType(getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)))
		}

		for _, element := range v.Elements {
			var plural string = "element"
			if element.Plural {
				plural = "has_many"
			}
			fieldType := genRubyFieldType(getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree))
			content += fmt.Sprintf("\t\t%s :%s, 'OTA::%s', tag: '%s'\n", plural, ToSnakeCase(genRubyFieldName(element.Name)), fieldType, element.Name)
		}
		content += "\tend\n"
		gen.StructAST[v.Name] = content
		gen.Field += fmt.Sprintf("\t%s\tclass %s\n\t\tinclude XmlMapper\n\n%s", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}

// RubyGroup generates code for group XML schema in Ruby language syntax.
func (gen *CodeGenerator) RubyGroup(v *Group) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		content := ""
		fieldName := genRubyFieldName(v.Name)
		if fieldName != v.Name {
			gen.ImportEncodingXML = true
			content += fmt.Sprintf("\t\ttag \"%s\"\n", v.Name)
		}
		for _, element := range v.Elements {
			var plural string
			if element.Plural {
				plural = ""
			}
			content += fmt.Sprintf("\t%s\t%s%s\n", ToSnakeCase(genRubyFieldName(element.Name)), plural, genRubyFieldType(getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)))

		}

		for _, group := range v.Groups {
			var plural string
			if group.Plural {
				plural = ""
			}
			content += fmt.Sprintf("\t%s\t%s%s\n", ToSnakeCase(genRubyFieldName(group.Name)), plural, genRubyFieldType(getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)))
		}
		content += "\tend\n"
		gen.StructAST[v.Name] = content
		gen.Field += fmt.Sprintf("\t%s\tclass %s\n\t\tinclude XmlMapper\n\n%s", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}

// RubyAttributeGroup generates code for attribute group XML schema in Ruby language
// syntax.
func (gen *CodeGenerator) RubyAttributeGroup(v *AttributeGroup) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		content := ""
		fieldName := genRubyFieldName(v.Name)
		if fieldName != v.Name {
			gen.ImportEncodingXML = true
			content += fmt.Sprintf("\t\ttag \"%s\"\n", v.Name)
		}
		for _, attribute := range v.Attributes {
			// content += fmt.Sprintf("\t%sAttr\t%s\t`xml:\"%s,attr%s\"`\n", ToSnakeCase(genRubyFieldName(attribute.Name)), genRubyFieldType(getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)), attribute.Name, optional)
			content += fmt.Sprintf("\t\tattribute :%s, 'OTA::%s', tag: '%s'\n", ToSnakeCase(genRubyFieldName(attribute.Name)), genRubyFieldType(getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)), attribute.Name)
			// fmt.Println(attribute.Name)
		}
		content += "\tend\n"
		gen.StructAST[v.Name] = content
		gen.Field += fmt.Sprintf("\t%s\tclass %s\n\t\tinclude XmlMapper\n\n%s", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}

// RubyElement generates code for element XML schema in Ruby language syntax.
func (gen *CodeGenerator) RubyElement(v *Element) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var plural string = genRubyFieldType(getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree))
		if v.Plural {
			plural = "Array"
		}
		content := fmt.Sprintf(" < %s; ", plural)
		gen.StructAST[v.Name] = content
		fieldName := genRubyFieldName(v.Name)
		gen.Field += fmt.Sprintf("\t%s\tclass %s%send\n", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}

// RubyAttribute generates code for attribute XML schema in Ruby language syntax.
func (gen *CodeGenerator) RubyAttribute(v *Attribute) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var plural string = genRubyFieldType(getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree))
		if v.Plural {
			plural = "Array"
		}
		content := fmt.Sprintf(" < %s; ", plural)
		gen.StructAST[v.Name] = content
		fieldName := genRubyFieldName(v.Name)
		gen.Field += fmt.Sprintf("\t%s\tclass %s%send\n", genFieldComment(fieldName, v.Doc, "#"), fieldName, gen.StructAST[v.Name])
	}
	return
}