package injector

import "unsafe"

// 这个文件下的方法就是用来将数据注入到给定的地址中
// int、bool、float等数据类型可以直接通过偏移得到的指针进行设置
// 结构体指针的底层数据其实是一个uintptr，但是在实际使用时是不知道需要设置的结构体类型的
// 也就没有办法使用通用的偏移指针进行设置，例如
// type A_struct struct {
// 	Name string
// }
// type B_Struct struct {
// 	A *A_struct
// }
// 如果只是简单地给B_Struct设置其中的A成员，只需要如下代码即可
// var b B_Struct
// var ptr = uintptr(unsafe.Pointer(&b))
// *((**A_struct)(unsafe.Pointer(ptr+0))) = &A_struct{
// 	Name: "hahah",
// }
// 但是，考虑其通用性，实际使用时不能每一个结构体指针都要写一段类似的代码来设置
// 所以考虑使用如下的汇编代码进行设置，该代码是在amd64架构的cpu上运行的，386和arm的代码目前没有实现
// 不过大致原理是一样的，具体可以看[Golang汇编](https://juejin.im/entry/6844903537688707079)
// 以及代码文件：injector_amd64.asm
// TEXT InjectStructPtr(SB), NOSPLIT, $0-16
//     MOVQ pos+8(SP), CX
//     MOVQ dat+16(SP), AX
//     MOVQ AX, (CX)
//     RET
// 由于汇编代码晦涩难懂，且需要基础
// 我就在其基础上，通过分析底层原理进一步使用go简化
// 因为以上的汇编代码也就是把二进制数拷入到对应地址
// 那么通过golang的uintptr也可以实现相同的操作
// 就是把结构体的指针--一般就是uintptr表示，拷入到存放指针的地址，也就是指针的指针指向的地方
// InjectStructPtr set *struct value
// func InjectStructPtr(pos unsafe.Pointer, data unsafe.Pointer) {
// 	*((*uintptr)(pos)) = uintptr(data)
// }

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
