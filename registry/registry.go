package registry

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
)

type TypeRegistry map[string]reflect.Type

var (
	Types = make(TypeRegistry)
)

func RegisterType(name string, typ reflect.Type) {
	if _, exists := Types[name]; exists {
		panic(fmt.Errorf("重複したインデックスを登録しようとしました: %s", name))
	}
	Types[name] = typ
}

func TypeByName(name string) reflect.Type {
	v, ok := Types[name]
	if !ok {
		logrus.Infof("TypeByName: %s", name)
	}
	return v
}

func TypeNameByInstance(instance interface{}) string {
	switch ins := instance.(type) {
	case map[string]interface{}:
		return reflect.TypeOf(ins).String()
	default:
		return reflect.TypeOf(ins).Elem().String()
	}
}

func TypeInstanceByName(name string) interface{} {
	return reflect.New(TypeByName(name)).Interface()
}
