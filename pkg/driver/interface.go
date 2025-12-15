package driver

// 核心注释 必须生成在 Handler 函数 的上方
type Handler interface {
	Summary() string
	Description() string
	Router() string
	Tags() []string
}

// 请求参数
type Param interface {
	Name() string
	ParamType() string
	DataType() string
	Required() bool
	Comment() string
}

// 响应数据
type Response interface {
	Code() int
	Data() interface{}
}
