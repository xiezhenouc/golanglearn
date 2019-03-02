# goalng方法的指针接收者 值接收者

## 1 概念
>方法的接收者

>因为大部分方法在被调用后都需要维护接收者值的状态，所以，一个最佳实践是，将方法的接收者声明为指针。

>但有例外情况：如果通过接口类型的值调用方法，规则有很大不同。`使用指针作为接收者声明的方法，只能在接口类型的值是一个指针的时候被调用。使用值作为接收者声明的方法，在接口类型的值为值或者指针时候，都可以被调用`

## 2 示例
## 2.1 方法接收者是值，接口类型是值和指针都OK
```
package main

import ()

type Feed struct {
	FID      uint64
	FContent string
}

type Result struct {
	Field   string
	Content string
}

type Matcher interface {
	Search(feed *Feed, searchTerm string) ([]*Result, error)
}

// 实现Matcher接口类型
type DefaultMatcher struct{}

func (d DefaultMatcher) Search(feed *Feed, searchTerm string) ([]*Result, error) {
	return nil, nil
}

func main() {
	feed := &Feed{}
	searchTerm := ""

	var dm DefaultMatcher

	// 使用值来调用方法, OK
	var matcher Matcher = dm
	_, _ = matcher.Search(feed, searchTerm)

	// 使用指针调用方法, OK
	var matcherPointer Matcher = &dm
	_, _ = matcherPointer.Search(feed, searchTerm)

}
```
```
构建可以执行
go build succ
```
## 2.1 方法接收者是指针，接口类型必须是指针
```
// 不同点是方法的接收者修改为指针
package main

import ()

type Feed struct {
	FID      uint64
	FContent string
}

type Result struct {
	Field   string
	Content string
}

// 实现Matcher接口类型
type Matcher interface {
	Search(feed *Feed, searchTerm string) ([]*Result, error)
}

type DefaultMatcher struct{}

func (d *DefaultMatcher) Search(feed *Feed, searchTerm string) ([]*Result, error) {
	return nil, nil
}

func main() {
	feed := &Feed{}
	searchTerm := ""

	var dm DefaultMatcher

	// 使用值来调用方法, WARONG!
	var matcher Matcher = dm
	_, _ = matcher.Search(feed, searchTerm)
	// 使用指针调用方法, OK
	var matcherPointer Matcher = &dm
	_, _ = matcherPointer.Search(feed, searchTerm)

}
```
```
// 结构，构建失败
go build fail
./hello.go:33:6: cannot use dm (type DefaultMatcher) as type Matcher in assignment:
	DefaultMatcher does not implement Matcher (Search method has pointer receiver)
```
