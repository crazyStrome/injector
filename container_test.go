package injector

import (
	"fmt"
	"testing"
	"unsafe"
)

type Abean struct {
	B    *Bbean `resource:"b"`
	Name string `data:"hahahha"`

	C int
}
type Bbean struct {
	Name string `data:"hehehhe"`
}

func TestInjector(t *testing.T) {
	type A_struct struct {
		Name string
	}
	type B_struct struct {
		a *A_struct
	}
	type Eface struct {
		Type *struct{}
		Data unsafe.Pointer
	}
	var b = &B_struct{}
	var bintr = (interface{})(b)
	var a = &A_struct{
		Name: "hsp",
	}
	var ainter = (interface{})(a)
	var aptr = (*Eface)(unsafe.Pointer(&ainter)).Data
	var beface_ptr = (*Eface)(unsafe.Pointer(&bintr)).Data
	InjectStructPtr(unsafe.Pointer(beface_ptr), aptr)
	fmt.Println(b.a)
}
func TestGetsingleton(t *testing.T) {
	var con = NewContainer()
	con.addSingleton("b", func() interface{} {
		return &Bbean{}
	})
	con.addSingleton("a", func() interface{} {
		return &Abean{}
	})
	var bean, err = con.getSingletonBeanByName("a")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(bean)
	var b, ok = bean.(*Abean)
	if ok {
		fmt.Println(b.B)
	}
	fmt.Println(b)
	con.getSingletonBeanByName("c")
	con.getSingletonBeanByName("b")
}

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
