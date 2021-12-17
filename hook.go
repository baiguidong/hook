package hook

/**
register 注册多个的话,会执行多次
rollback 使用的是para的拷贝,非深度拷贝
根据需要,如果想使用当时的值 values 设置值类型,如果需要引用最后的使用引用类型
*/
import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

var (
	hookModel   sync.Map
	defaultHook = "default"
)

type Hook struct {
	hookMap sync.Map
}

func (h *Hook) Register(hookkey *Hookkey, f IHookRun) {
	hookInfo := fmt.Sprintf("Hook Regester %s", hookkey.Name)
	hookName, ok := f.(IHookName)
	if ok {
		hookInfo += fmt.Sprintf("->%s", hookName.Name())
	}
	log.Println(hookInfo)
	val, ok := h.hookMap.Load(hookkey)
	if ok {
		vals, ok := val.(HooksRun)
		if ok {
			vals.addHandle(f)
			h.hookMap.Store(hookkey, vals)
		}
	} else {
		hooks := HooksRun{}
		hooks.addHandle(f)
		h.hookMap.Store(hookkey, hooks)
	}
}
func (h *Hook) RunHook(hookkey *Hookkey, c *Context) {
	if c.abort {
		return
	}
	value, ok := h.hookMap.Load(hookkey)
	if !ok {
		return
	}
	hooks, ok := value.(HooksRun)
	if !ok {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			c.AbortErr(fmt.Errorf(string(debug.Stack())))
		}
	}()
	for n, hook := range hooks.hooks {
		namehead := fmt.Sprintf("Run [%s->][%d]:", hookkey.Name, n)
		hookname, nameok := hook.(IHookName)
		if nameok {
			namehead = fmt.Sprintf("Run [%s->%s][%d]:", hookkey.Name, hookname.Name(), n)
		}
		hook.Run(c)
		if c.abort {
			log.Println(namehead + "abort!")
			break
		}
		log.Println(namehead + "ok!")
		hookclean, cleanok := hook.(IHookClean)
		if cleanok {
			c.rollback.addHandle(hookclean, hookkey, c)
		}
	}
}

func getHook(name string) *Hook {
	v, ok := hookModel.Load(name)
	if ok {
		return v.(*Hook)
	}
	h := &Hook{}
	hookModel.Store(name, h)
	return h
}

type Hookkey struct {
	Name string
}
type Context struct {
	Ctx      context.Context
	param    sync.Map
	abort    bool
	err      error
	rollback HooksClean
	HookName string
}

// Clone param 的简单拷贝  用于 rollback
// TODO 非深度拷贝,使用需要注意
func (c *Context) clone() *Context {
	res := Context{}
	c.param.Range(func(key, value interface{}) bool {
		res.param.Store(key, value)
		return true
	})
	return &res
}

// GetPara 	根据key 获取业务参数
func (c *Context) GetParam(key interface{}) interface{} {
	val, ok := c.param.Load(key)
	if ok {
		return val
	}
	log.Printf("GetParam nil %v", key)
	return nil
}

// SetPara 设置业务参数
func (c *Context) SetParam(key, value interface{}) {
	c.param.Store(key, value)
}

// DelPara 删除业务参数
func (c *Context) DelPara(key interface{}) {
	c.param.Delete(key)
}

// IsAbort 是否结束
func (c *Context) IsAbort() bool {
	return c.abort
}

// IsAbortErr 是否错误结束
func (c *Context) IsAbortErr() bool {
	return c.abort && c.err != nil
}

// Abort 退出执行流程 并且rollback已经执行的hook(需要提供Clean方法)
func (c *Context) AbortErr(err error) {
	c.rollback.runClean()
	c.abort = true
	c.err = err
	log.Printf("AbortErr:%s", err.Error())
}
func (c *Context) AbortOk() {
	c.abort = true
	c.err = nil
}
func (c *Context) AbortResult(err error, key, val interface{}) {
	c.AbortErr(err)
	c.SetParam(key, val)
}

// Err 返回Hook 错误
func (c *Context) Err() error {
	return c.err
}
func (c *Context) OK() bool {
	return c.err == nil
}

// IHookRun hook 必须实现的接口 用于hook的执行
type IHookRun interface {
	Run(*Context)
}

// HooksRun 一个Key 对应HooksRun(多个hook)
type HooksRun struct {
	hooks []IHookRun
}

// HooksRun 添加hook 到 HooksRun
func (h *HooksRun) addHandle(f IHookRun) {
	h.hooks = append(h.hooks, f)
}

// IHookClean 需要RollBack的hook 需要实现的方法
type IHookClean interface {
	Clean(*Context)
}

// HooksClean 存放已经执行成功 并且实现Clean 方法的hook
type HooksClean struct {
	hookcs   []IHookClean
	keynames []*Hookkey
	params   []*Context
}

// AddHandle 添加 HookClean以及key 和参数拷贝 到 HooksClean
func (h *HooksClean) addHandle(f IHookClean, k *Hookkey, c *Context) {
	h.hookcs = append(h.hookcs, f)
	h.keynames = append(h.keynames, k)
	h.params = append(h.params, c.clone())
}

// RunClean 出现错误需要执行的 rollback
func (h *HooksClean) runClean() {
	for n := len(h.hookcs) - 1; n >= 0; n-- {
		namehead := fmt.Sprintf("Clean [%s->]", h.keynames[n].Name)
		hookname, nameok := h.hookcs[n].(IHookName)
		if nameok {
			namehead = fmt.Sprintf("Clean [%s->%s]", h.keynames[n].Name, hookname.Name())
		}
		log.Println(namehead)
		h.hookcs[n].Clean(h.params[n])
	}
}

// IHookName 实现返回函数名称的接口 hook 实现该接口后 方便日志查看
type IHookName interface {
	Name() string
}

// Regester register hook to map
func Register(key *Hookkey, f IHookRun) {
	RegisterName(defaultHook, key, f)
}
func RegisterName(name string, key *Hookkey, f IHookRun) {
	getHook(name).Register(key, f)
}

// RunHook 根据key 运行已经注册的 hook
func RunHook(key *Hookkey, c *Context) {
	name := c.HookName
	if name == "" {
		name = defaultHook
	}
	getHook(name).RunHook(key, c)
}

// PrintMap hookMap 输出
func PrintMap(name string) {
	if name == "" {
		name = defaultHook
	}
	getHook(name).hookMap.Range(func(key, value interface{}) bool {
		log.Printf("%+v -> %T -> %d\n", key, value, len(value.(HooksRun).hooks))
		return true
	})
}
