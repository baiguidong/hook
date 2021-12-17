# Hook 使用
## 第一步 需要注册 hook 函数
```golang
	// bird
	Register(birdkey, bird{})
	Register(birdkey, bird{})
	// tiger
	Register(tigerkey, tiger{})
	// lion
	Register(lionkey, lion{})
```
## 第二步 需要准备调用链参数
```golang
    param := &Context{}
	param.SetParam("name", "my is bird")
```
## 第三部 调用hook 运行
```golang
    // 根据 hookkey 调用hook
	RunHook(birdkey, param)
	if param.IsAbort() {
		// 如果调用链 abort
		return
	}
```
# 完整实例

```golang
package main

import (
	"errors"
	"fmt"
)
// 鸟
type bird struct {
}
func (p bird) Name() string {
	return "my is bird hook"
}
func (p bird) Run(t *Context) {
	fmt.Println("[run] my param name --> ", t.GetParam("name").(string))
}
func (p bird) Clean(t *Context) {
	fmt.Println("[rollback] my param name <-- ", t.GetParam("name").(string))
}
// 老虎
type tiger struct {
}
func (p tiger) Run(t *Context) {
	fmt.Println("[run] my param name --> ", t.GetParam("name").(string))
}
// 狮子
type lion struct {
}
func (p lion) Name() string {
	return "my is lion hook"
}
func (p lion) Run(t *Context) {
	fmt.Println("[run] my param name --> ", t.GetParam("name").(string))
}

var (
	birdkey  = &Hookkey{"bird"}
	tigerkey = &Hookkey{"tiger"}
	lionkey  = &Hookkey{"lion"}
)

// 初始化注册 hook 程序启动时需要注册
func initregister() {
	// bird
	Register(birdkey, bird{})
	Register(birdkey, bird{})
	// tiger
	Register(tigerkey, tiger{})
	// lion
	Register(lionkey, lion{})
}
func main() {
	// 注册hook
	initregister()
	// 初始化调用链参数
	param := &Context{}
	param.SetParam("name", "my is bird")
	// 根据 hookkey 调用hook
	RunHook(birdkey, param)
	if param.IsAbort() {
		// 如果调用链 abort
		return
	}
	// 调整调用链参数
	param.SetParam("name", "my is tiger")
	// 根据 hookkey 调用hook
	RunHook(tigerkey, param)
	if param.IsAbort() {
		// 如果调用链 abort
		return
	}
	// 调整调用链参数
	param.SetParam("name", "my is lion")
	// 根据 hookkey 调用hook
	RunHook(lionkey, param)
	if param.IsAbort() {
		// 如果调用链 abort
		return
	}
	// 主动abort 触发 rollback
	param.AbortErr(errors.New("test abort"))
	if param.IsAbort() {
		fmt.Println(param.Err().Error())
	}
    //打印注册hook信息
	PrintMap("")
}

```

# 运行效果

```golang
//由于bird 实现了rollback 所以主动出发abort的时候 bird 会运行rollback
[run] my param name -->  my is bird
[run] my param name -->  my is bird
[run] my param name -->  my is tiger
[run] my param name -->  my is lion
[rollback] my param name <--  my is bird
[rollback] my param name <--  my is bird
test abort
```