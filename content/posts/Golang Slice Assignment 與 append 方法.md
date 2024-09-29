---
title: Golang Slice Assignment 與 append 方法
date: 2022-08-16
tags:
- Golang
---

## slice 資料結構

Golang 底層使用 [SliceHeader](https://pkg.go.dev/reflect#SliceHeader) 去描述運行時的 slice

```go
type SliceHeader struct {
    Data uintptr
    Len  int
    Cap  int
}
```

對應一個 slice 具備的三個屬性：

- Len - slice 的長度
- Cap - 底層陣列大小
- Data - 底層 array **第一個元素**的**指標**

slice 有點類似 java 的 ArrayList，len 就等同於 size，表示 list 當中的元素數量，而 cap 則對應 ArrayList 的 capacity，也就是底層陣列的大小。

> 之所以特別強調 slice 所記錄的底層陣列指標是「陣列第一個元素的指標」，是因為要提醒不要忘了陣列的本質就是一串連續的記憶體空間，儲存第一個元素指標（位址）後即可藉由索引及元素大小來計算出其他索引確切的記憶體位址。

## Slice Assignment

```go
s1 := make([]int, 0)

s2 := s1
```

執行 `s2 := s1` 時，雖然是複製出另一個 slice（記憶體位置不同），但兩者的指標仍是同一個，都指向相同的陣列（起始位置）

> 除非其中一個 slice 在 append 時，發現原本陣列空間不夠，因此建另一個新的陣列，才會造成兩者所指向的底層陣列不同，有關 append 的 reallocate 機制，參考：[https://go.dev/blog/slices](https://go.dev/blog/slices)
> 


所以當新增一個如下的 slice：

```go
s1 := make([]int, 5, 10)

fmt.Println(s1) // [0, 0, 0, 0, 0]

fmt.Println(len(s1), cap(s1), &s1[0]) // 5 10 0xc0000ac002
```

等同新建一個大小為 10 的陣列，並在陣列填入 `5` 個 `int` 類型的 zero value，再把 slice 的 len 設為 5， cap 設為 10，並且紀錄陣列第一個元素的指標。

用圖示表示：

![Untitled](images/go-slice-1.png)

所以當我們要取超過 slice 長度(len)的元素

```go
fmt.Println(s1[5])
```

會獲得一個 `runtime error: index out of range [5] with length 5` 的 panic

## Slice Assignment + append 方法的非預期行為

當要往一個 len 為 0 的 slice 當中添加元素時，會需要使用 append 方法，該方法會回傳一個更新後的 slice。通常都會這樣使用：

```go
nums := make([]int, 0, 8)
nums = append(nums, 1) // append 過後的結果直接賦值給 nums 變數
```

延伸上面提到的 slice assignment 概念，當另外一個變數 `newNums` 也接收 append 後回傳的新 slice，兩個 slice (`nums` & `newNums`) 所指向底層陣列相同時，以下程式碼會印出什麼？

```go
nums := make([]int, 0, 8)
nums = append(nums, 1)
nums = append(nums, 2)
nums = append(nums, 3)
nums = append(nums, 4)
nums = append(nums, 5)

newNums := append(nums, 6)
newNums = append(newNums, 7)
nums = append(nums, 7)

fmt.Println(newNums)
fmt.Println(nums)
```

結果會是：

```go
[1 2 3 4 5 7 7]
[1 2 3 4 5 7]
```

如果覺得意外的話，我們就一個個來看究竟發生什麼事吧。

首先，在建立 `newNums` 之前， `nums` 跟底層陣列長這樣：

![Untitled](images/go-slice-2.png)

接著到了 `newNums := append(nums, 6)` 這行，建立新的 slice 並且加了一個新的元素 6 到陣列：

![Untitled](images/go-slice-3.png)

這時可能有人會問，既然底層陣列多了一個元素，然後原本的 `nums` 也指向這個陣列，那 `nums` 不就也會多一個元素嗎？

**但其實動到的是底層陣列而不是 `num` 這個 slice**，因為 `nums` 本身的 `len` 並沒有改變（改變的是 `newNums`的 `len` ），剛剛前面有提到，當要取超過 slice 長度(len)的元素，會拋出例外。也就是說，在 `nums` 這個 slice 雖然本身底層陣列已經遭到改動了，但**因爲我們都是透過 slice 去對底層陣列取值的 ，而 `nums` 的 `len` 還是 5，根本無法 reach 到 index 為 5 的元素**。

繼續往下到 `newNums = append(newNums, 7)` 這一行，底層陣列又多了一個 7：

![Untitled](images/go-slice-4.png)

接著到 `nums = append(nums, 7)` 這行，由於 `nums` slice 最後一個元素的 index 等於 4，所以 append 方法是會將陣列 index 4+1=5 的位置放入 7 這個值，所以就造成 index 5 原本的值被取代：

![Untitled](images/go-slice-5.png)

驗證 `nums` index 5 和 `newNums` index 5 的指標（記憶體位置）確實相同：

```go
newNums := append(nums, 6)
fmt.Println(newNums[5], &newNums[5]) // 6 0xc000128068 ->這時候還是 6

newNums = append(newNums, 7)
fmt.Println(newNums[6], &newNums[6]) // 7 0xc000128070

nums = append(nums, 7)
fmt.Println(nums[5], &nums[5]) // 7 0xc000128068 ->被上面那行改成 7
fmt.Println(newNums[5], &newNums[5]) // 7 0xc000128068 ->和 nums 為同一個記憶體位置

fmt.Println(nums) // [1 2 3 4 5 7]
fmt.Println(newNums) // [1 2 3 4 5 7 7] -> index 5 的值已被覆蓋
```

總結，slice 可以視作在陣列之上的一層“片段資訊”，藉由這個片段，我們可以對底層陣列取值、增刪改元素，同時也允許多個片段向同個陣列操作，不同片段取的索引範圍可能也不同。但也是因此必須非常小心，因為只要底層陣列一被改動，所有參考該陣列的 slice 都可能會受到影響。