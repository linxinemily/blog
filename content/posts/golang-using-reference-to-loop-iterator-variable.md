---
title: Golang 常見錯誤 - 在迴圈當中引用迭代變數(iterator variable)
date: 2023-02-21
tags:
- Golang
---

前幾天閱讀到[這篇官方 wiki](https://github.com/golang/go/wiki/CommonMistakes) ，其中的 [Using reference to loop iterator variable](https://github.com/golang/go/wiki/CommonMistakes#using-reference-to-loop-iterator-variable)，也就是關於在迴圈當中引用迭代變數常犯的錯誤。當中有個範例一開始讓我不是很理解，後來終於搞懂，所以才想說筆記下來，也順便釐清一下思路。

首先，第一個範例滿好理解的：

```go
func main() {
	var out []*int
	for i := 0; i < 3; i++ {
		out = append(out, &i)
	}
	fmt.Println("Values:", *out[0], *out[1], *out[2])
	fmt.Println("Addresses:", out[0], out[1], out[2])
}
```

輸出的結果會是：

```
Values: 3 3 3
Addresses: 0x40e020 0x40e020 0x40e020
```

原因是，由於 append 到  `out`  這個 slice 的參數是  `&i` ，傳入的是一個 `int 類型的變數指標`（ `out` 的型別也是一個 `存放多個 int 類型的變數指標 的 slice`），所以每個迭代都是把同一個變數指標 append 到 `out` 裡面，因此當迭代到最後一次時，該變數指標裡面存放的值已經變成 `3`  ，所以印出來就是都是 `3` 。

而解決的辦法就是把 `i` 的值複製到另一個新變數上：

```go
for i := 0; i < 3; i++ {
  i := i // Copy i into a new variable.
 	out = append(out, &i)
 }
```

這樣一來， 就能保證每次迭代當中的 `&i` 就是當前迭代所創造的新變數的指標，不會導致非預期的結果，也讓原本只能存活在 for scope 當中的變數 `i` 能夠在迴圈之外被引用，不會在迴圈結束後就被回收。

接下來的另一個範例，就是讓我一開始不是很能理解的地方：

```go
func main() {
	var out [][]int
	for _, i := range [][1]int{{1}, {2}, {3}} {
		out = append(out, i[:])
	}
	fmt.Println("Values:", out)
}
```

以上程式的輸出是：

```
Values: [[3] [3] [3]]
```

但如果當只改動一個地方：

```go
func main() {
	var out [][]int
	for _, i := range [][]int{{1}, {2}, {3}} { // 把長度 1 拿掉
		out = append(out, i[:])
	}
	fmt.Println("Values:", out)
}
```

以上的輸出就變成預期的結果：

```
Values: [[1] [2] [3]]
```

..等等，不是才差一個字嗎？實際搞懂後，差一個字但其實差很多，因為這兩個範例當中的迭代變數類型完全不同。

原本的範例當中，被迭代的 slice 裏面放的是 `長度為 1 且內容物為 int 類型` 的陣列： `[][1]int{{1}, {2}, {3}}` ，而後面更動的版本，被迭代的 slice 裡面放的是另一個 slice： `[][]int{{1}, {2}, {3}}` 。因為在 Golang 當中，宣告陣列和 slice 的寫法很像，差別在於陣列是需要指定長度的，不指定長度的話會變成 slice。

```go
var a [1]int // 帶指定長度會是陣列
var b []int // 不帶長度會是 slice
```

而且陣列和其他純值類型一樣是屬於 value type。也就是當把陣列賦值給另一個變數時，等同拷貝一份一樣的陣列進去新的變數，原始陣列不會受到該新陣列的更動影響：

```go
arr := [1]int{1}
arr2 := arr
arr2[0] = 2
fmt.Println(arr)
// output: [1] 
```

因此，在兩個範例當中 `i[:]` 所代表的意義就不同。第一個範例中的 `i` 是陣列，`i[:]`  代表一個引用 `i` 這個陣列的 slice （slice 底層指向的陣列指標 = `&i`），雖然每個迭代中都會創造一個新的 slice，但這些 slice 底層所引用的都是同一個變數指標 。

到這邊就可以看出，其實這個狀況跟最一開始 `i` 為 int 類型變數的範例一樣， 由於每次迭代時， `i` 的值都一直在替換，直到迴圈結束， `i` 的值停留在最後一次迭代當中被賦予的值。

而第二個範例中， `i[:]` 是代表將原先 `i` 這個 slice 複製出一份 slice，而基於複製 slice 的原理，這兩個 slice 底層都會指向同一個陣列（複製的是 slice 的 header，包含陣列的(第一個元素的)指標、長度的值）

> What exactly is this slice variable? It’s not quite the full story, but for now think of a slice as a little data structure with two elements: a length and a pointer to an element of an array. You can think of it as being built like this behind the scenes:
> 
> 
> ```
> type sliceHeader struct {
>     Length        int
>     ZerothElement *byte
> }
> ```
> 

但 slice 本身仍是一個 value

> It’s important to understand that even though a slice contains a pointer, it is itself a value. Under the covers, it is a struct value holding a pointer and a length. It is *not*
 a pointer to a struct.
> 

所以當下一次迭代時， `i` 的內容換成下一個 slice，會再次複製 `i` 的值，也就是 header 的部分給新的 slice，每次迭代中產生的新 slice 都和迭代當下的 slice 有相同的 header，每個複製也都引用和複製對象相同的底層陣列。整個流程始終沒有去引用到 `i` 的指標，也就不會有上面範例的問題。

總結，迭代變數在迭代同時，值也會不斷改變，如果在迴圈中有引用到迭代變數，可能會有非預期的行為。

## References

[CommonMistakes](https://github.com/golang/go/wiki/CommonMistakes)

[research!rsc: Go Data Structures](https://research.swtch.com/godata)

[Arrays, slices (and strings): The mechanics of 'append' - The Go Programming Language](https://go.dev/blog/slices)