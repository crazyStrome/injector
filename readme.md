---
title: injector依赖注入库-golang实现
updated: 2020/11/10
comments: true
tags:
- Golang
- 反射
- 依赖注入
categories:
- Golang
cover: https://cdn2.hubspot.net/hubfs/202339/golang.png

---



# injector

[github/injector](https://github.com/crazyStrome/injector)

injector是一个依赖注入的库，把需要依赖注入的对象：一般是结构体，slice等容器型对象和其对应的beanName添加到Container中，然后在使用时从Container中通过beanName获取所需要的bean，并通过定义在结构体Field的tag来进行依赖注入，最后返回给调用者。

在golang自带的json解析库中，遇到不可导出的field是无法设置value的，injector使用偏移指针来设置value，这样一来，不可导出的field也可以通过tag设置。

TODO：通过beanType来获取对应的bean

TODO：protoType的依赖注入

TODO：slice的依赖注入

2020.10.13：目前实现了基本数据类型和*struct的依赖注入，只有singleton

2020.11.10：目前实现了通过指针设置struct的field，而不是使用低效率的reflect.Value

##  使用demo

###  field注入（可导出和不可导出）

使用如下标签进行依赖注入：

* data：将数据注入field，injector会根据field的类型对data数据进行类型转换
* resource：将resource对应的对象名称注入field，前提是resource对应的bean和field的类型相同
* autowired：有`type`和`name`两种选择，如果是`name`，就是通过field的名字生成beanName在container中查找并注入field；如果是`type`，就是通过field的类型生成beanName并在container查找然后注入field。

```go
func main() {

	type Bbean struct {
		Id       int
		Name     string `data:"lihua"`
		unexport string `data:"this is a unexport string"`
	}
	type Abean struct {
		B      *Bbean `resource:"b"`
		BBBBBB *Bbean `autowired:"type"`
		Name   string `data:"xiaoming"`

		C int

		unexport bool `data:"true"`
	}
	// 生成一个container
	var con = injector.NewContainer()

	// 注册bean
	var err = con.Registe("a", func() interface{} {
		return &Abean{}
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = con.Registe("b", func() interface{} {
		return &Bbean{
			Id: 1001,
		}
	})
	if err != nil {
		log.Fatalln(err)
	}

	err = con.Registe("bbean", func() interface{} {
		return &Bbean{
			Id: 1002,
		}
	})

	a, err := con.GetBeanByName("a")
	if err != nil {
		log.Fatalln(err)
	}
	if abean, ok := a.(*Abean); ok {
		fmt.Printf("the bean get from container as a is %T:::%+v\n", abean, abean)
		fmt.Printf("the inner bean from ABean as B is %T:::%+v\n", abean.B, abean.B)
		fmt.Printf("the inner bean from ABean as BBBBBB is %T:::%+v\n", abean.BBBBBB, abean.BBBBBB)
	}

	b, err := con.GetBeanByName("b")
	if err != nil {
		log.Fatalln(err)
	}
	if bbean, ok := b.(*Bbean); ok {
		fmt.Printf("the bean get from container as a is %T:::%+v\n", bbean, bbean)
	}

	bbb, err := con.GetBeanByName("bbean")
	if err != nil {
		log.Fatalln(err)
	}
	if bbbean, ok := bbb.(*Bbean); ok {
		fmt.Printf("the bean get from container as bbean is %T:::%+v\n", bbbean, bbbean)
	}
}
```

输出：

```bash
the bean get from container as a is *main.Abean:::&{B:0xc00010c450 BBBBBB:0xc00010c510 Name:xiaoming C:0 unexport:true}
the inner bean from ABean as B is *main.Bbean:::&{Id:1001 Name:lihua unexport:this is a unexport
string}
the inner bean from ABean as BBBBBB is *main.Bbean:::&{Id:1002 Name:lihua unexport:this is a unexport string}
the bean get from container as a is *main.Bbean:::&{Id:1001 Name:lihua unexport:this is a unexport string}
the bean get from container as bbean is *main.Bbean:::&{Id:1002 Name:lihua unexport:this is a unexport string}
```

###  循环依赖

代码：

```go

type B struct {
	Name   string `data:"BBean"`
	Abean  *A     `resource:"a"`
	AAbean *A     `autowired:"type"`
}
type A struct {
	Num   int    `json:"tag" data:"2020"`
	Name  string `data:"ABean"`
	bbean *B     `require:"true" resource:"b"`
	exist bool   `data:"true"`
	a     int
}

func main() {
	var con = injector.NewContainer()
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
	var sa, ok = ainter.(*A)
	if ok {
		fmt.Printf("ptr of sa is 0x%x, content of sa is %+v\n", uintptr(unsafe.Pointer(sa)), sa)
		fmt.Printf("ptr of sa.Bbean is 0x%x, content of sa.Bbean is %+v\n", uintptr(unsafe.Pointer(sa.bbean)), sa.bbean)
	}
}
```

输出：

```bash
ptr of sa is 0xc00010c360, content of sa is &{Num:2020 Name:ABean bbean:0xc000004520 exist:true a:0}
ptr of sa.Bbean is 0xc000004520, content of sa.Bbean is &{Name:BBean Abean:0xc00010c360 AAbean:0xc00010c360}
```

##  代码思路

injector使用三级缓存实现，如下所示。

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
```

singletonBeans 是一级缓存，保存已经初始化和依赖注入完成的对象， 使用singletonInCreation进行标记
earlySingletonBeans 是二级缓存，保存已经初始化但是没有完成依赖注入的对象， 使用singletonAlreadyCreated进行标记
singletonFactories 是三级缓存，保存工厂函数，用于生成bean

一级缓存和二级缓存保存的都是可以使用的bean，用于解决循环依赖问题。

实现思路如下：

![image.png](https://i.loli.net/2020/11/10/SHcqXo1kEwYGNtK.png)

## 依赖注入

依赖注入主要是通过反射注入对象，详情见[知乎-通过反射注入对象](https://zhuanlan.zhihu.com/p/279861676)

###  基本类型注入

injector.go文件中的api。

```go
// InjectInt set int value
func InjectInt(pos unsafe.Pointer, data int) {
	*((*int)(pos)) = data
}

// InjectBool set bool value
func InjectBool(pos unsafe.Pointer, data bool) {
	*((*bool)(pos)) = data
}

// InjectString set string value
func InjectString(pos unsafe.Pointer, data string) {
	*((*string)(pos)) = data
}

// InjectFloat64 set float64 value
func InjectFloat64(pos unsafe.Pointer, data float64) {
	*((*float64)(pos)) = data
}

// InjectFloat32 set float32 value
func InjectFloat32(pos unsafe.Pointer, data float32) {
	*((*float32)(pos)) = data
}

// InjectStructPtr set *struct value
func InjectStructPtr(pos unsafe.Pointer, data unsafe.Pointer) {
	*((*uintptr)(pos)) = uintptr(data)
}

```

这个文件下的方法就是用来将数据注入到给定的地址中，int、bool、float等数据类型可以直接通过偏移得到的指针进行设置，结构体指针的底层数据其实是一个uintptr，但是在设置结构体指针的时候就比较复杂。例如：

```go
type A_struct struct {

  Name string

}

type B_Struct struct {

  A *A_struct

}
```

如果只是简单地给B_Struct设置其中的A成员，只需要如下代码即可

```go
var b B_Struct

var ptr = uintptr(unsafe.Pointer(&b))

*((**A_struct)(unsafe.Pointer(ptr+0))) = &A_struct{

  Name: "hahah",

}
```

但是，考虑其通用性，实际使用时不能每一个结构体指针都要写一段类似的代码来设置。

所以考虑使用如下的汇编代码进行设置，该代码是在amd64架构的cpu上运行的，386和arm的代码目前没有实现。不过大致原理是一样的，具体汇编的知识可以看[Golang汇编](https://juejin.im/entry/6844903537688707079)，以及代码文件：injector_amd64.asm

```asm
TEXT InjectStructPtr(SB), NOSPLIT, $0-16

  MOVQ pos+8(SP), CX

  MOVQ dat+16(SP), AX

  MOVQ AX, (CX)

  RET
```

以上汇编代码晦涩难懂，不过从底层原理的角度可以使用go简化。以上汇编代码就是把AX中的数据copy到CX中的数据作为地址指向的内存中。例如：AX中存放的是0x1000，CX中存放的是0x2000，那么这段汇编就是把0x2000地址处的数据设置为0x1000。

在golang中，通过uintptr也可以实现相同的操作，就是把结构体的指针（uintptr表示）data，拷入到pos存放的地址指向的内存空间中。

```go
// InjectStructPtr set *struct value

func InjectStructPtr(pos unsafe.Pointer, data unsafe.Pointer) {

  *((*uintptr)(pos)) = uintptr(data)

}
```

###  循环依赖注入

获取bean的伪代码：
解决了循环依赖

```go
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

示例：

定义如下两个结构体：

```go
type B struct {
	Name   string `data:"BBean"`
	Abean  *A     `resource:"a"`
}
type A struct {
	Num   int    `json:"tag" data:"2020"`
	Name  string `data:"ABean"`
	bbean *B     `require:"true" resource:"b"`
}
```

结构体A依赖结构体B，结构体B也依赖于A。

1. 调用者获取A的实例
2. 一级缓存和二级缓存都没有A的实例
3. 调用三级缓存中的工厂方法实例化A
4. 对A的field进行注入，发现需要B
    1. 调用者获取B的实例
    2. 一级缓存和二级缓存都没有B的实例
    3. 调用三级缓存中的工厂方法实例化B
    4. 对B的field进行注入，发现需要A
        1. 在二级缓存中查找到A的实例
        2. 返回A给上级调用者
    5. B注入完成，添加到一级缓存中，返回B给上级调用者
5. 获取到B，注入到A
6. 将A添加到一级缓存中，返回A给上级调用者