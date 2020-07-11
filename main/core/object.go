package core

// RedisObject 是对特定类型的数据的包装
type RedisObject struct {
	ObjectType int         // 类型定义
	Ptr        interface{} // 实际值
}

const ObjectTypeString = 1 // 指定类型的常量值, 例如String为1

// CreateObject 创建特定类型的object结构
func CreateObject(t int, ptr interface{}) (o *RedisObject) {
	o = new(RedisObject)
	o.ObjectType = t
	o.Ptr = ptr
	return
}
