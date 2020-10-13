package injector

import (
	"fmt"
	"testing"
)

type Abean struct {
	B    *Bbean `resource:"b"`
	Name string `data:"hsp"`

	C int
}
type Bbean struct {
	Name string `data:"wy"`
}

func TestGetsingleton(t *testing.T) {
	var con = NewContainer()
	// con.addSingleton("b", func() interface{} {
	// 	return &Bbean{}
	// })
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
