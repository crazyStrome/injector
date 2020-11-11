package injector

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

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
	// logrus.SetOutput(os.Stderr)
	//设置最低loglevel
	logrus.SetLevel(logrus.ErrorLevel)
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

	// populatedSingletonField map[string]map[string]bool
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
		// populatedSingletonField: make(map[string]map[string]bool),
	}
}

// getSingleton used for getting a singleton from three cache
// if beanType == nil, get the bean by name
// TODO: get bean by type, for now get beanName from type to get bean is used instaed
// if beanName == "", get the bean by name from beanType
// for example, beanType == main.A, the name generated is "a"
// 调用该方法返回的是一级缓存或者二级缓存中的bean
// 如果返回的是二级缓存中的bean，则会附带返回一个ErrBeanLoadNotTotal
func (con *Container) getSingleton(beanName string, beanType reflect.Type) (bean interface{}, err error) {
	if beanName == "" && beanType == nil {
		return bean, ErrWrongParameter
	}
	if beanType != nil && beanType.Kind() == reflect.Ptr {
		beanType = beanType.Elem()
	}
	// get the beanName
	if len(beanName) == 0 || beanName == "" {
		// 本来应该是通过bean的类型在container获取对应的bean的
		// 这个功能留待之后实现
		// TODO
		// 目前通过beanType获取beanName，然后通过beanName获取bean
		beanName = getBeanNameByType(beanType)
		log.Infof("func getSingleton: beanName is null, so use type %s get the beanName %s\n", beanType, beanName)

	}
	// 从一级缓存中获取，如果一级缓存中存在则直接返回，此时的bean已经是注入完成的
	if bean, ok := con.singletonBeans[beanName]; ok {
		log.Infof("getSinglton: %s is get from singletonBeans\n", beanName)
		return bean, nil
	}
	// 如果bean在二级缓存中，也直接返回，此时注入是半完成的，所以也返回一个error
	if bean, ok := con.earlySingletonBeans[beanName]; ok {
		// judge if all populated
		// removed from earlySingletonBeans
		// and add to singletonBeans
		log.Infof("getSinglton: %s is get from earlySingletonBeans\n", beanName)
		return bean, ErrBeanLoadNotTotal
	}
	// 如果一级缓存和二级缓存中都没有需要的bean
	// 就需要去三级bean工厂中找对应的工厂方法
	// 如果没有对应的工厂方法，就抛出异常
	beanFunc, ok := con.singletonFactories[beanName]
	if !ok {
		log.Errorf("getSinglton: %s didn't exists in container\n", beanName)
		return nil, ErrNoSuchBeanRegisted
	}

	bean = beanFunc()
	// 因为是singleton，所以bean工厂方法使用之后就可以删除了
	delete(con.singletonFactories, beanName)

	// 现在的bean需要注入依赖，所以在singletonInCreation中设置标记
	// 然后把bean添加到二级缓存中
	con.singletonInCreation[beanName] = true
	con.earlySingletonBeans[beanName] = bean

	log.Infof("getSinglton: %s is get from singletonFactories, and add to earlySingletonBeans: %+v\n", beanName, con.earlySingletonBeans)
	return bean, ErrBeanLoadNotTotal
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

	var bean = beanFunc()
	var beanType = reflect.TypeOf(bean)
	// TODO
	// 依赖注入到slice
	if beanType.Kind() != reflect.Ptr || (beanType.Elem().Kind() != reflect.Struct && beanType.Elem().Kind() != reflect.Slice) {
		log.Printf("addSingleton: beanFunc must return *struct")
		return ErrWrongParameter
	}

	// 如果已经注册到三级工厂缓存中了，返回error
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
	return con.getSingletonBeanByName(beanName)
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
		return con.getSingleton(beanName, nil)
	}

	// 如果在二级缓存中，也直接返回，不过有一个error
	if con.singletonInCreation[beanName] {
		log.Infof("getSingletonBeanByName: this bean named '%s' is in creation\n", beanName)
		return con.getSingleton(beanName, nil)
	}

	// if the bean didn't registe in container, as didn't be in factory
	// return error
	if _, ok := con.singletonFactories[beanName]; !ok {
		return nil, ErrNoSuchBeanRegisted
	}
	// first initialization
	var bean, err = con.getSingleton(beanName, nil)
	if err != nil && err == ErrNoSuchBeanRegisted {
		return bean, err
	}

	// bean population
	con.populateSingleton(beanName)
	if con.singletonAlreadyCreated[beanName] {
		log.Infof("getSingletonBeanByName: after population, this bean named '%s' is alread created\n", beanName)
		return bean, nil
	}
	log.Infof("getSingletonBeanByName: this bean named '%s' is in creation, not load totally, but returns as result\n", beanName)
	return bean, ErrBeanLoadNotTotal
}

// eface表示bean转为interface的底层结构
type eface struct {
	typ *struct{}
	obj unsafe.Pointer
}

func (con *Container) populateSingleton(beanName string) {

	// 这个时候的bean已经在二级缓存里了
	var bean, _ = con.getSingleton(beanName, nil)

	// 获取到的bean都是*...类型的对象，所以需要Elem()来获取其指向的内容
	var beanType = reflect.TypeOf(bean).Elem()

	// 此时获取到的bean是interface{}，底层表示的就是一个eface
	// 所以通过设置eface.obj就可以设置对象的field
	var beanStartPtr = (*eface)(unsafe.Pointer(&bean)).obj

	var populateSum = beanType.NumField()
	// var populatesCount = 0

	log.Infof("populateSingleton: bean '%s' of type '%v' is being populated\n", beanName, beanType)

	for i := 0; i < populateSum; i++ {

		// 不需要进行计数，遇到基本类型直接注入
		// 遇到结构体的话或者其他对象例如A就通过container的标记查找，
		// 如果在一级缓存或者二级缓存中找到依赖B，则直接注入，然后继续下一个成员
		// 如果还没有实例化对象B，则需要实例化并初始化注入依赖，让需要注入的对象B中的依赖注入完成
		// 此时B也放在了一级缓存中，B也注入完成
		// 再注入到原来的对象A中，之后A也放在了一级缓存中了
		// 这个也解决了循环依赖

		var field = beanType.Field(i)
		// 获取field上的tag
		var tag = field.Tag
		if len(tag) == 0 || tag == "" {
			continue
		}

		var fieldPtr = unsafe.Pointer(uintptr(beanStartPtr) + field.Offset)

		var fieldType = field.Type

		switch fieldType.Kind() {
		case reflect.Int:
			var data = tag.Get("data")
			var d, err = strconv.Atoi(data)
			if err != nil {
				log.Println("convert error: ", err)
				continue
			}
			log.Printf("populateSingleton: convert data '%s' to int\n", data)
			InjectInt(fieldPtr, d)
		// case reflect.Uint:
		case reflect.Float32:
			var data = tag.Get("data")
			var d, err = strconv.ParseFloat(data, 32)
			if err != nil {
				log.Println("convert error: ", err)
				continue
			}
			log.Printf("populateSingleton: convert data '%s' to float32\n", data)
			InjectFloat32(fieldPtr, float32(d))
		case reflect.Float64:
			var data = tag.Get("data")
			var d, err = strconv.ParseFloat(data, 64)
			if err != nil {
				log.Println("convert error: ", err)
				continue
			}
			log.Printf("populateSingleton: convert data '%s' to float64\n", data)
			InjectFloat64(fieldPtr, d)
		case reflect.String:
			var data = tag.Get("data")
			log.Printf("populateSingleton: convert data '%s' to string, actually convertion isn't needed\n", data)
			InjectString(fieldPtr, data)
		case reflect.Bool:
			var data = tag.Get("data")
			var d, err = strconv.ParseBool(data)
			if err != nil {
				log.Println("convert error: ", err)
				continue
			}
			log.Printf("populateSingleton: convert data '%s' to bool\n", data)
			InjectBool(fieldPtr, d)
		case reflect.Ptr:
			if fieldType.Elem().Kind() == reflect.Struct {
				// fmt.Println(fieldType)
				// 这部分是获取beanName的
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
				log.Printf("populateSingleton: get beanName from resource '%s', fieldName '%s' or type '%s'\n", tag.Get("resource"), fieldType.Name(), fieldType.Elem().Name())
				// bean, err := con.getSingletonBeanByName(beanName)
				// 因为需要另一个对象作为依赖，可以直接调用getSingletonBeanByName
				bean, err := con.getSingletonBeanByName(beanName)
				if err == ErrNoSuchBeanRegisted {
					continue
				}
				var btype = reflect.TypeOf(bean)
				if btype.Kind() == reflect.Ptr && btype.Elem().Kind() == fieldType.Elem().Kind() {
					// return reflect.ValueOf(bean)
					var beanPtr = (*eface)(unsafe.Pointer(&bean)).obj
					InjectStructPtr(fieldPtr, beanPtr)
				}

			}
		}

	}
	// remove from earlySIngletonBeans
	// add to singletonBeans
	con.singletonAlreadyCreated[beanName] = true
	delete(con.singletonInCreation, beanName)

	con.singletonBeans[beanName] = bean
	delete(con.earlySingletonBeans, beanName)

}

// func ifFieldUnExported(fieldName string) bool {
// 	return fieldName != "" && fieldName[0] >= 'a' && fieldName[0] <= 'z'
// }
