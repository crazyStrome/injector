package injector

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func init() {
	// log.SetFormatter(&log.JSONFormatter{})
	// log.SetFormatter(&log.TextFormatter{
	// 	DisableColors: true,
	// 	FullTimestamp: true,
	// })
	logrus.SetFormatter(&logrus.JSONFormatter{})
	//设置output,默认为stderr,可以为任何io.Writer，比如文件*os.File
	logrus.SetOutput(os.Stdout)
	//设置最低loglevel
	logrus.SetLevel(logrus.InfoLevel)
}

// Scope show the bean scope
type Scope uint8

const (
	// Prototype for scope,
	// each time a bean is get different
	Prototype Scope = iota
	// Singleton for scope
	// each time a bean is get the same
	Singleton
)

var (
	// ErrBeanLoadNotTotal is a error reprents a bean didn't load totally
	ErrBeanLoadNotTotal = errors.New("bean load not totally")

	// ErrWrongParameter is a error reprents a func didn't have right parms
	ErrWrongParameter = errors.New("wrong parameters: beanName or beanType is needed")

	// ErrNoSuchBeanRegisted presents the bean isn't registed before
	ErrNoSuchBeanRegisted = errors.New("no such bean is registed before")

	// ErrBeanExists presents the bean is registed before
	ErrBeanExists = errors.New("already registed the bean")
)

// type beanDefination struct {
// 	beanType  reflect.Type
// 	beanFunc  func() interface{}
// 	beanName  string
// 	beanScope Scope
// }

// Container is the container for depency inject
type Container struct {
	singletonBeans          map[string]interface{}
	earlySingletonBeans     map[string]interface{}
	singletonFactories      map[string]func() interface{}
	singletonInCreation     map[string]bool
	singletonAlreadyCreated map[string]bool

	populatedSingletonField map[string]map[string]bool
}

// NewContainer return a new container
func NewContainer() *Container {
	return &Container{
		// beans already created and populated
		singletonBeans: make(map[string]interface{}),
		// beans already created but didn't be populated
		earlySingletonBeans: make(map[string]interface{}),
		// bean factory, need to be created
		singletonFactories: make(map[string]func() interface{}),

		// marker for the beans created totally
		singletonInCreation: make(map[string]bool),
		// marker for the beans in population
		singletonAlreadyCreated: make(map[string]bool),

		// marker for population in field counting
		populatedSingletonField: make(map[string]map[string]bool),
	}
}

// getSingleton used for getting a singleton from three cache
// if beanType == nil, get the bean by name
// if beanName == "", get the bean by name from beanType
// for example, beanType == main.A, the name generated is "a"
func (con *Container) getSingleton(beanName string, beanType reflect.Type) (bean interface{}, err error) {
	if beanName == "" && beanType == nil {
		return bean, ErrWrongParameter
	}
	if beanType != nil && beanType.Kind() == reflect.Ptr {
		beanType = beanType.Elem()
	}
	// get the beanName
	if len(beanName) == 0 || beanName == "" {
		beanName = getBeanNameByType(beanType)
		log.Infof("func getSingleton: beanName is null, so use type %s get the beanName %s\n", beanType, beanName)

	}
	if bean, ok := con.singletonBeans[beanName]; ok {
		log.Infof("getSinglton: %s is get from singletonBeans\n", beanName)
		return bean, nil
	}
	if bean, ok := con.earlySingletonBeans[beanName]; ok {
		// judge if all populated
		// removed from earlySingletonBeans
		// and add to singletonBeans
		log.Infof("getSinglton: %s is get from earlySingletonBeans\n", beanName)
		return bean, ErrBeanLoadNotTotal
	}
	beanFunc, ok := con.singletonFactories[beanName]
	if !ok {
		log.Errorf("getSinglton: %s didn't exists in container\n", beanName)
		return nil, errors.New("no such bean registed, beanName:" + beanName)
	}

	bean = beanFunc()
	delete(con.singletonFactories, beanName)
	con.singletonInCreation[beanName] = true
	con.earlySingletonBeans[beanName] = bean
	log.Infof("getSinglton: %s is get from singletonFactories, and add to earlySingletonBeans: %+v\n", beanName, con.earlySingletonBeans)
	return bean, nil
}

// this func is used for transform type to name
// for example,type is *main.A, the name is a
func getBeanNameByType(beanType reflect.Type) string {
	if beanType.Kind() == reflect.Ptr {
		beanType = beanType.Elem()
	}
	var typeName = beanType.String()
	var lastDotIndex = strings.LastIndex(typeName, ".")
	beanName := typeName[lastDotIndex+1:]
	var tmp = []byte(beanName)
	if tmp[0] >= 'A' {
		tmp[0] = tmp[0] - 'A' + 'a'
	}
	beanName = string(tmp)
	return beanName
}

// Registe is used for adding bean to container
func (con *Container) Registe(beanName string, beanFunc func() interface{}) error {
	return con.addSingleton(beanName, beanFunc)
}

// thie func is used for add singleton to singletonFactories
func (con *Container) addSingleton(beanName string, beanFunc func() interface{}) error {
	if beanFunc == nil {
		log.Printf("addSingleton: wrong parameters...beanName or beanFunc is needed")
		return ErrWrongParameter
	}
	if beanName == "" {
		beanName = getBeanNameByType(reflect.TypeOf(beanFunc()))
	}
	// TODO
	// add a type assert
	// if beanFunc can't produce a *struct type, return error
	// DONE
	var bean = beanFunc()
	var beanType = reflect.TypeOf(bean)
	if beanType.Kind() != reflect.Ptr || beanType.Elem().Kind() != reflect.Struct {
		log.Printf("addSingleton: beanFunc must return *struct")
		return ErrWrongParameter
	}

	if _, ok := con.singletonFactories[beanName]; ok {
		return ErrBeanExists
	}

	con.singletonFactories[beanName] = beanFunc
	log.Infof("addSingleton: bean named '%s' with beanFunc '%s' is add to container\n", beanName, reflect.TypeOf(beanFunc))
	log.Infoln("addSingleton: the maps of container", con)
	return nil
}

// GetBeanByName is used for get bean
func (con *Container) GetBeanByName(beanName string) (interface{}, error) {
	var bean, err = con.getSingletonBeanByName(beanName)
	if err != nil && err == ErrBeanLoadNotTotal {
		con.populateSingleton(beanName)
	}
	if con.singletonAlreadyCreated[beanName] {
		return bean, nil
	}
	return bean, ErrBeanLoadNotTotal
}

// this func is used for get bean by name
// @params beanName: the name of bean to be found
// @return interface{}: the bean found
//			error: if didn't have the bean, return ErrNoSuchBeanRegisted
//				   if didn't initailization totally, return ErrBeanLoadNotTotal
func (con *Container) getSingletonBeanByName(beanName string) (interface{}, error) {
	// if the bean is in the cache, return directly
	if con.singletonAlreadyCreated[beanName] {
		log.Infof("getSingletonBeanByName: this bean named '%s' is already created\n", beanName)
		var bean, _ = con.getSingleton(beanName, nil)
		return bean, nil
	}

	if con.singletonInCreation[beanName] {
		log.Infof("getSingletonBeanByName: this bean named '%s' is in creation\n", beanName)
		var bean, _ = con.getSingleton(beanName, nil)
		return bean, ErrBeanLoadNotTotal
	}

	// if the bean didn't registe in container, as didn't be in factory
	// return error
	if _, ok := con.singletonFactories[beanName]; !ok {
		return nil, ErrNoSuchBeanRegisted
	}
	// first initialization
	var bean, _ = con.getSingleton(beanName, nil)

	// bean population
	con.populateSingleton(beanName)
	if con.singletonAlreadyCreated[beanName] {
		log.Infof("getSingletonBeanByName: after population, this bean named '%s' is alread created\n", beanName)
		return bean, nil
	}
	log.Infof("getSingletonBeanByName: this bean named '%s' is in creation, not load totally, but returns as result\n", beanName)
	return bean, ErrBeanLoadNotTotal
}

// for example,
// type A struct {
//	b B
//}
// b would be load to a as a object of A
func (con *Container) populateSingleton(beanName string) {
	// if
	// if con.singletonAlreadyCreated[beanName] {
	// 	return
	// }
	if _, ok := con.populatedSingletonField[beanName]; !ok {
		con.populatedSingletonField[beanName] = make(map[string]bool)
	}
	var bean, _ = con.getSingleton(beanName, nil)

	var beanType = reflect.TypeOf(bean).Elem()
	var beanValue = reflect.ValueOf(bean).Elem()

	var populateSum = beanType.NumField()
	// var populatesCount = 0

	log.Infof("populateSingleton: bean '%s' of type '%v' is being populated\n", beanName, beanType)

	for i := 0; i < populateSum; i++ {

		var fieldName = beanType.Field(i).Name
		var fieldType = beanType.Field(i).Type

		var fieldValue = beanValue.Field(i)
		log.Infof("populateSingleton: populate field '%s' from bean %s\n", fieldName, beanName)
		// if this field is populated, continue
		if con.populatedSingletonField[beanName][fieldName] {
			continue
		}
		// marked as populated
		con.populatedSingletonField[beanName][fieldName] = true

		var tag = beanType.Field(i).Tag

		var dataValue = con.getBaseTypeDataByTag(tag, fieldType)
		log.Printf("populateSingleton: get dataValue as '%+v' from getBaseTypeDataByTag\n", dataValue)

		if fieldValue.CanSet() {
			log.Printf("populateSingleton: populate field '%+v' with type of '%s' \n", fieldName, fieldValue.Type())

			fieldValue.Set(dataValue)
		} else {
			// TODO
			// unexported needed to use unsafe.Pointer
			// or for struct such as abean, use the func SetAbean()
		}

	}
	// remove from earlySIngletonBeans
	// add to singletonBeans
	if len(con.populatedSingletonField[beanName]) == populateSum {
		con.singletonAlreadyCreated[beanName] = true
		delete(con.singletonInCreation, beanName)

		con.singletonBeans[beanName] = bean
		delete(con.earlySingletonBeans, beanName)

		delete(con.populatedSingletonField, beanName)
	}
}
func ifFieldUnExported(fieldName string) bool {
	return fieldName != "" && fieldName[0] >= 'a' && fieldName[0] <= 'z'
}

// this func is used for transfor string type data to BaseType data
// for example, "100"==>100 of int
func (con *Container) getBaseTypeDataByTag(tag reflect.StructTag, fieldType reflect.Type) reflect.Value {
	log.Printf("getBaseTypeDataByTag: get data from tag '%s' by the type of '%s'\n", tag, fieldType)
	var empty = reflect.Zero(fieldType)
	if fieldType.Kind() == reflect.Ptr {
		empty = reflect.New(fieldType.Elem())
	}

	if tag == "" {
		return empty
	}

	var data = tag.Get("data")

	if (len(data) == 0 || data == "") && (fieldType.Kind() != reflect.Ptr) {
		return empty
	}

	switch fieldType.Kind() {
	case reflect.Int:
		var d, err = strconv.Atoi(data)
		if err != nil {
			log.Println("convert error: ", err)
			return empty
		}
		log.Printf("getBaseTypeDataByTag: convert data '%s' to int\n", data)
		return reflect.ValueOf(d)
	// case reflect.Uint:
	case reflect.Float32:
		var d, err = strconv.ParseFloat(data, 32)
		if err != nil {
			log.Println("convert error: ", err)
			return empty
		}
		log.Printf("getBaseTypeDataByTag: convert data '%s' to float32\n", data)
		return reflect.ValueOf(float32(d))
	case reflect.Float64:
		var d, err = strconv.ParseFloat(data, 64)
		if err != nil {
			log.Println("convert error: ", err)
			return empty
		}
		log.Printf("getBaseTypeDataByTag: convert data '%s' to float64\n", data)
		return reflect.ValueOf(d)
	case reflect.String:
		log.Printf("getBaseTypeDataByTag: convert data '%s' to string, actually convertion isn't needed\n", data)
		return reflect.ValueOf(data)
	case reflect.Bool:
		var d, err = strconv.ParseBool(data)
		if err != nil {
			log.Println("convert error: ", err)
			return reflect.New(fieldType)
		}
		log.Printf("getBaseTypeDataByTag: convert data '%s' to bool\n", data)
		return reflect.ValueOf(d)
	case reflect.Ptr:
		if fieldType.Elem().Kind() == reflect.Struct {
			fmt.Println(fieldType)
			var beanName = tag.Get("resource")
			if beanName == "" {
				if tag.Get("autowired") == "type" {
					beanName = fieldType.Elem().Name()

				} else {
					beanName = fieldType.Name()
				}
				if beanName[0] >= 'A' && beanName[0] <= 'Z' {
					var tmp = []byte(beanName)
					tmp[0] = tmp[0] - 'A' + 'a'
					beanName = string(tmp)
				}
			}
			log.Printf("getBaseTypeDataByTag: get beanName from resource '%s', fieldName '%s' or type '%s'\n", tag.Get("resource"), fieldType.Name(), fieldType.Elem().Name())
			// bean, err := con.getSingletonBeanByName(beanName)
			bean, err := con.getSingletonBeanByName(beanName)
			if err == nil || err == ErrBeanLoadNotTotal {
				// log.Errorln("convert error: ", err)
				// con.populateSingleton(beanName)
				var btype = reflect.TypeOf(bean)
				if btype.Kind() == reflect.Ptr && btype.Elem().Kind() == fieldType.Elem().Kind() {
					return reflect.ValueOf(bean)
				}
			}

		}
	}
	return empty
}
