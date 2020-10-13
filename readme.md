# injector
这是一个依赖注入的库
2020.10.13：目前实现了基本数据类型和*struct的依赖注入，只有singleton

## Container 

```golang
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
```
singletonBeans 是一级缓存，保存已经初始化和依赖注入完成的对象， 使用singletonInCreation进行标记
earlySingletonBeans 是二级缓存，保存已经初始化但是没有完成依赖注入的对象， 使用singletonAlreadyCreated进行标记
singletonFactories 是三级缓存，保存工厂函数，用于生成bean

##  依赖注入思路
获取bean的伪代码：
解决了循环依赖
```
func getBean(beanName string) interface{} {
    //如果在一级或者二级缓存，直接返回bean
    if in the cache(beanName) {
        return bean
    }
    //没有找到bean, 通过工厂方法获取
    // 通过三级缓存的工厂方法获取未完成依赖注入的bean
    // 并把bean添加到二级缓存中，并从三级缓存中删除
    var bean = beanFactory.getBean() //对应于 getSingleton

    // 依赖注入 对应于populateSingleton函数
    for i := 0; i < bean.NumField(); i ++ {
        // 需要使用getBaseTypeDataByTag解析需要注入的数据，包括对象
        if bean is base type {
            bean.Field(i).Set(xxx)
        } else {
            // 获取bean
            bean.Field(i).Set(getBean(field.Name))
        }
    }
    //bean注入完成，将bean添加到一级缓存，并从二级缓存中删除
}
```


## tag 注解

* resource:name 通过name查找bean
* autowired:name,type 通过name或者type查找bean，默认name
* data:100 通过data注解给基本类型赋初值

## 示例
```golang
type B struct {
	Name   string `data:"hsp"`
	Abean  *A     `resource:"a"`
	AAbean *A     `autowired:"type"`
}
type A struct {
	Num   int    `json:"tag" data:"2020"`
	Name  string `data:"Ahsp"`
	Bbean *B     `require:"true" resource:"b"`
	Exist bool   `data:"true"`
	a     int
}

func TestCicurlation(t *testing.T) {
	var con = NewContainer()
	con.Registe("a", func() interface{} {
		return &A{}
	})
	con.Registe("b", func() interface{} {
		return &B{}
	})
	var ainter, err = con.GetBeanByName("a")
	if err != nil {
		fmt.Println(err)
		return
	}
	// fmt.Println(a)
	var sa, ok = ainter.(*A)
	if ok {
		fmt.Printf("ptr of sa is 0x%x, content of sa is %+v\n", uintptr(unsafe.Pointer(sa)), sa)
		fmt.Printf("ptr of sa.Bbean is 0x%x, content of sa.Bbean is %+v\n", uintptr(unsafe.Pointer(sa.Bbean)), sa.Bbean)
	}
}

```

result:
```bash
ptr of sa is 0xc00011bdd0, content of sa is &{Num:2020 Name:Ahsp Bbean:0xc000005120 Exist:true a:0}
ptr of sa.Bbean is 0xc000005120, content of sa.Bbean is &{Name:hsp Abean:0xc00011bdd0 AAbean:0xc00011bdd0}
```