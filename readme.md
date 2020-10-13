# injector
这是一个依赖注入的库
2020.10.13：目前实现了基本数据类型和*struct的依赖注入，只有singleton

## tag 注解

* resource:name 通过name查找bean
* autowired:name,type 通过name或者type查找bean，默认name
* data:100 通过data注解给基本类型赋初值

## 依赖注入
通过getBaseTypeDataByTag从tag获取相关的内容从而生成Value，注入需要的field中