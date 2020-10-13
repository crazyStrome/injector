## tag 注解

* resource:name 通过name查找bean
* autowired:name,type 通过name或者type查找bean，默认name
* data:100 通过data注解给基本类型赋初值

## 依赖注入
普通类型使用unsafe.Pointer获取
结构体类型通过Value.Set
*struct使用Value.Set
*普通类型用unsafe.Pointer