---
title: Golang struct 轉 JSON 遇到欄位沒有值的問題
date: 2023-03-27
tags:
- Golang
aliases:
- /posts/golang-marshal-json-omitempty
---

要將 Go 的 struct 轉成 JSON 格式時，通常都會藉由官方提供的標準函式庫 `encoding/json` 去實現，也會借助在 struct 當中為特定欄位定義 `json` tag 去標示輸出成 JSON 格式時的命名和規則。在欄位值存在的情況下基本上不會有什麼問題，但如果欄位沒有值，就有可能會輸出非預期結果（簡稱：踩坑）。

首先必須先釐清一件事，所謂的「沒有值」準確來說是指什麼？nil？空陣列？還是空字串？我們都知道在 Golang 當中，不同類型的資料有著不同的「零值(Zero Value)」，比如 string 的零值是空字串、int 的零值是 0。然而「沒有值」比較偏向一個籠統的說法，也不管程式語言的定義，反正就是沒有資料的意思。*又或者說以使用者角度來看，因為不管欄位資料裡面是空字串或是 NULL，對他們來說就都是沒資料*。

以下會以這個 struct 為範例，分別來看這兩種不同類型(reference v.s. value type)的資料欄位，在面臨沒有資料時會輸出什麼結果。

```go
type Person struct {
  Name    string   `json:"name"`
  Hobbies []string `json:"hobbies"`
}
```


## Reference Type - Slice

當欄位類型為 slice 的時候，在不同情況下的「沒有值」，轉換出來的結果也會有所不同。

1.當 `Hobbies` 是一個已初始化過後的空 slice：

```go
// 初始化方式一：
hobbies := make([]string, 0)
// 初始化方式二：
// hobbies := []string{}
// 都是初始化一個長度/容量皆為 0 的 slice

emmie := Person{
	"Emmie", hobbies, 
}

jsonData, _ := json.Marshal(emmie)
fmt.Println(string(jsonData))
```

輸出：

```json
{"name":"Emmie","hobbies":[]}
```

2.當 `Hobbies` 是一個只有宣告變數，沒有初始化的 slice：

```go
var hobbies []string

emmie := Person{
  "Emmie", hobbies,
}

jsonData, _ := json.Marshal(emmie)
fmt.Println(string(jsonData))
```

輸出：

```json
{"name":"Emmie","hobbies":null}
```

之所以會有這種差異是因為在 Golang 中，**當宣告一個變數但是沒有初始化值的話，其預設值會是該類型的零值**，而對於 slice、map、channel 這樣的類型(reference type)來說，零值是 nil。

在第一個範例中，slice 變數會被初始化為一個長度及容量皆為 0 的空 slice，而不是 nil。所以 `hobbies` 在宣告時已經被初始化為一個空的 slice，因此 `emmmie` 的 `hobbies` 欄位會是一個空的 slice，它的 JSON 表示就是 `[]`。

然而，在第二個範例中，`hobbies` 變數僅有被宣告，所以預設值為 nil，而不是一個空的 slice。在 JSON 轉換期間，nil slice 會被轉換成 JSON 的 null 值。

如果想在 `hobbies` 欄位為 null 或空陣列時（也就是 slice 為 nil 或 空 slice 的狀態下），不要輸出該欄位 key，像這樣：

```json
{"name":"Emmie"}
```

那就要修改 struct 當中的定義，為 `hobbies` 加上 `omitempty` 標籤：

```go
Hobbies []string `json:"hobbies,omitempty"`
```

統整一下，所以在將 struct 轉換成 JSON 時，如果有欄位的類型是 slice ：

| 狀況 | 輸出 JSON 的值 |
| --- | --- |
| 未初始化（nil slice） | `null` |
| 已初始化（空 slice） | `[]` |
| 未初始化（nil slice）但加上 `omitempty` 標籤 | key 直接消失 |
| 已初始化（空 slice）但加上 `omitempty` 標籤 | key 直接消失 |

可以發現 slice 類型（reference type）的欄位加上 `omitempty` ，不管其值是 nil（零值） 或是空 slice，key 都會消失。

## Value Type - String

那如果當欄位類型是 string、int 等 value type 呢？

繼續使用上面的範例 struct，在還沒加上 `omitempty` tag 時：

```go
type Person struct {
  Name    string   `json:"name"`
  Hobbies []string `json:"hobbies,omitempty"`
}
```

```go
emmie := Person{}

jsonData, _ := json.Marshal(emmie)
fmt.Println(string(jsonData))
```

輸出：

```json
{"name":""}
```

如果把 `name` 也加上 `omitempty` tag：

```go
type Person struct {
  Name    string   `json:"name,omitempty"`
  Hobbies []string `json:"hobbies,omitempty"`
}
```

輸出：

```go
{}
```

string 未初始化情況下，預設為零值空字串， key 也消失了。

但如果我們不想讓 key 直接消失，想要輸出 null 的話該怎麼辦？方法是將 struct 當中的 `string` 改成指標類型 `*string` （因為指標類型的零值也是 nil）且不需要 `omitempty` tag：

```go
type Person struct {
  Name    *string  `json:"name"`
  Hobbies []string `json:"hobbies,omitempty"`
}
```

輸出：

```json
{"name":null}
```

再統整一下：

| 類型 | 狀況 | 輸出 JSON 的值 |
| --- | --- | --- |
| `string` | 未初始化（空字串） | `""` |
| `string` | 未初始化（空字串）但加上 `omitempty` 標籤 | key 直接消失 |
| `*string` | 未初始化 （nil）| `null` |

## 總結
* 在不同類型中，根據零值不同，輸出的 JSON 值也不同。
* 然而在相同類型中，輸出的 JSON 值也有不同的可能性。例如剛剛提到的 slice 類型，初始化後的空 slice 會輸出空陣列，而未初始化為零值的 nil slice 會輸出 null。
* `omitempty` tag 會影響 key 值的存留。

只要注意這幾點，就可以避免在輸出 JSON 時得到非預期的資料類型/結構。