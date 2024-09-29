---
title: "GraphQL Dataloader 在 Golang 當中的打開方法與原理解析（下）"
date: 2023-06-04T17:44:11+08:00
description: 接續上篇介紹完 GraphQL Dataloader 在 Golang 當中的用法，接著探討 Dataloader 原始碼的部分，了解其背後運作原理。
tags:
- GraphQL
- Golang
aliases:
- /posts/go-gqlgen-and-dataloader-2
---

接續上一篇文章：[GraphQL Dataloader 在 Golang 當中的打開方法與原理解析（上）]({{< ref "/posts/golang-graphql-dataloader-1.md" >}})。

在進入原始碼之前，先複習一下 use case。首先從 resolver 開始看起：
```go
// IconURL is the resolver for the icon_url field.
func (r *roomFeatureResolver) IconURL(ctx context.Context, obj *model.RoomFeature) (*string, error) {
  f, _ := storage.GetFeature(ctx, *obj.ID)
  return f.IconUrl, nil
}
```
當中呼叫了我們的自定義方法 `storage.GetFeature`：
```go
func GetFeature(ctx context.Context, featureID int) (*model.RoomFeatureWithData, error) {
  //...
  thunk := loaders.FeatureLoader.Load(ctx, DataLoader.StringKey(fmt.Sprintf("%d", featureID)))
  result, err := thunk()
  //...
  return result.(*model.RoomFeatureWithData), nil
}
```

重點就是這裡使用到的 `loaders.FeatureLoader.Load` 方法。 
> `FeatureLoader` 雖為自定義屬性，但是透過 DataLoader 套件提供的 `*DataLoader.Loader` 建構子 `DataLoader.NewBatchedLoader` 所創建出，所以本身就具備了 Load 方法。

在深入探究 `Load` 方法之前，先釐清一下這個 `GetFeature` 方法什麼時候會被執行到。回到一開始的案例，
當 Client 進行如下 Query 時：
```graphql
query {
  content(hotel_id: 1) {
    rooms {
      features {
        id
        icon_url
      }
    }
  }
}
```

如果有 n 個 rooms，每個 room 有 m 個 features，總共會執行 n * m 次 `IconURL` 來取得每個 feature 的 icon URL，也就是執行 m * n 次 `GetFeature`。而在每個 `GetFeature` 方法當中又呼叫了 `loaders.FeatureLoader.Load` 方法，透過回傳的函數取得 feature 資料。


單純以使用上的角度，可以把 `Load` 方法當成是一個黑盒子，當我們向他索取需要的資料時，他就會回傳給我們。


想像如果這些方法都是同步被執行的，假設第一次傳入 `GetFeature` 的 feature id 為 1，此時要向 DataLoader 取 id 為 1 的 feature 資料，照理來說應該還沒有辦法取得，因為這時候 DataLoader 尚未蒐集到所有 rooms 的 features，（要搜集完所有 id 才會向資料庫查詢），所以應該會暫時 block 住，等待所有 id 蒐集完後才取得所有資料，才能取得 id 1 的資料。可想而知，這個功能背後一定是有**非同步**的支持才有可能辦到。

所以到底 DataLoader 是在什麼時候蒐集到所有 rooms 的 features 每個 id 的？又是在什麼時候呼叫向資料庫查詢的方法？就來看看這個 `Load` 方法到底做了什麼吧！

*然後因為這個方法有點長，所以我會按照順序分段貼上來，但不會全部貼，比較非核心功能的部分也會暫時省略。有興趣的讀者可以自行去看原始碼。*

首先創建一個接收結果的 channel，以及一個 result 變數，接著試圖到 `Loader` 的 cache 當中尋找 key 對應的值存不存在，有的話就直接返回值。
```go
//github.com/graph-gophers/DataLoader@v5.0.0+incompatible/DataLoader.go
func (l *Loader) Load(originalContext context.Context, key Key) Thunk {
  ctx, finish := l.tracer.TraceLoad(originalContext, key)

  c := make(chan *Result, 1)
  var result struct {
    mu    sync.RWMutex
    value *Result
  }

  l.cacheLock.Lock()
  if v, ok := l.cache.Get(ctx, key); ok {
    defer finish(v)
    defer l.cacheLock.Unlock()
    return v
  }
  //...下面續
}

```

返回值是 `Thunk` 類型，其實就是一個這樣形狀的 function：
```go
type Thunk func() (interface{}, error)
```
接著，當在 cache 當中找不到對應值時，就新建一個 thunk 函數，當中會使用上面建立的 result 變數，如果 result 裡的值為空，就一直等待直到有資料傳進 channel， 然後將 result 的值更新為傳入的 value。最後寫入 cache。
``` go
func (l *Loader) Load(originalContext context.Context, key Key) Thunk {
  // ...接續上面
  thunk := func() (interface{}, error) {
    result.mu.RLock() //上讀鎖，避免在讀取資料時，同時有其他線程想更新該資料
    resultNotSet := result.value == nil
    result.mu.RUnlock() // 釋放讀鎖

    if resultNotSet {
        result.mu.Lock() //上獨占鎖，避免在更新資料時，同時有其他線程想更新該資料
        if v, ok := <-c; ok { // block 住直到 c 有 result 傳入
          result.value = v
        }
        result.mu.Unlock() //釋放獨佔鎖
    }
    result.mu.RLock()
    defer result.mu.RUnlock()
    return result.value.Data, result.value.Error
  }
  defer finish(thunk)

  l.cache.Set(ctx, key, thunk)
  l.cacheLock.Unlock()

  //...下面續
}
```

再來，新增一個 batchRequest，並檢查 loader 當中的 curBatcher 存在與否，不存在就創建新的。這個 curBatcher 內部會有執行自定義方法——也就是在 NewBatchedLoader 時傳入的 `featureReader.GetFeature` 方法——的邏輯。

接著另開一個 goroutine 執行 curBatcher 的 `batch` 方法之後，又開了一個 goroutine 執行 loader 的 sleeper 方法。

然後把 batchRequest 傳入到了 curBatcher 的 input channel，最後回傳 thunk 函數：

```go
type batchRequest struct {
  key     Key
  channel chan *Result
}

type batcher struct {
  input    chan *batchRequest
  batchFn  BatchFunc
    //...
}

func (l *Loader) Load(originalContext context.Context, key Key) Thunk {
    //...接續上面

  // req 包含了要查詢的 key 和 等待接收結果的 channel 
  req := &batchRequest{key, c}

  l.batchLock.Lock()

  if l.curBatcher == nil {
    l.curBatcher = l.newBatcher(l.silent, l.tracer)

    go l.curBatcher.batch(originalContext)

    l.endSleeper = make(chan bool)
    go l.sleeper(l.curBatcher, l.endSleeper)
  }

  l.curBatcher.input <- req

  // 如果有額外設定 batchCap，可以限制一次最多可以接受的 req 數量
  if l.batchCap > 0 { // 但我們並沒有設定，所以 batchCap 為 0，不會進入這邊
    //...
  }
  l.batchLock.Unlock()

  return thunk
}
```

這時，發現邏輯進到了 `curBatcher.batch` 方法當中，所以緊接著來看這段方法：

逐一接收 input 的資料之後，終於看到了執行 `b.batchFn(ctx, keys)` 也就是我們自定義的資料庫查詢方法！
然後就是把查詢到的資料，依序放入每個 request 的 channel，再關閉 channel，如此一來 thunk 函數當中等待該 channel 傳回 result 那一行就會取得值而繼續往下執行。

```go
func (b *batcher) batch(originalContext context.Context) {
  var (
    keys     = make(Keys, 0)
    reqs     = make([]*batchRequest, 0)
    items    = make([]*Result, 0)
    panicErr interface{}
  )

    // keys 的順序 等同 reqs 的順序
  for item := range b.input {
    keys = append(keys, item.key)
    reqs = append(reqs, item)
  }

  ctx, finish := b.tracer.TraceBatch(originalContext, keys)
  defer finish(items)

  func() {
        // ...
    items = b.batchFn(ctx, keys) // 執行自定義的資料庫查詢方法
  }()

    // ...

    // 因為自定義方法當中也有依照 keys 的順序回傳 result，且 keys 的順序 等同 reqs 的順序
    // 所以這樣放入順序不會亂
  for i, req := range reqs { 
    req.channel <- items[i]
    close(req.channel)
  }
}
```

但是別忘了，`b.input` 也是一個 channel，這邊又使用了 for..range 語法來遍歷 channel，照理來說，除非別的 goroutine 當中有把這個 channel 給關了，否則應該會造成 deadlock 才對。而在哪裡會將 input 這個 channel 關閉呢？

總共有兩個地方都在 Load 方法當中，一個是當 `l.batchCap > 0` 時，如果符合指定條件(執行的 req 數量等於 batchCap 數量時)，會呼叫
`l.curBatcher.end()` 關閉 input chan。

```go
if l.batchCap > 0 { 
  //...
  if l.count == l.batchCap {
    l.curBatcher.end() // 直接終止接收 req，不再繼續等待
    close(l.endSleeper) // 關閉等待特定一段時間的 channel
    l.reset()
  }
}
```

另一個是位於 `go l.sleeper(l.curBatcher, l.endSleeper)` 這一行，所以還得再看一下這個方法做了什麼才行。

```go
func (l *Loader) sleeper(b *batcher, close chan bool) {
  select {
  // 只有在 Load 方法中進入 `l.batchCap > 0` 的 if 區塊時，會對 close channel 傳送訊號
  // 這邊收到訊號表示已經提前關閉了 input channel，也不需繼續等待下去
  case <-close:
    return
  case <-time.After(l.wait):
  }

  l.batchLock.Lock()
  b.end()  // <-----here

    //...
}

func (b *batcher) end() {
  if !b.finished {
    close(b.input)
    b.finished = true
  }
}
```

可以看到，當等待時間一到(`time.After(l.wait)`收到訊號後)，就會往下執行到 `b.end()`，也就是關閉 input channel 的方法，而等待的時間 `l.wait` 預設值是 16ms：

```go
func NewBatchedLoader(batchFn BatchFunc, opts ...Option) *Loader {
  loader := &Loader{
    batchFn:  batchFn,
    inputCap: 1000,
    wait:     16 * time.Millisecond, // <-----here
  }
  //...
}
```
也就是說，如果 loader 的 input channel 在 16ms 之後都沒有再收到 request 的話，就會自動關閉了。

## 總結

看完以上的原始碼之後，算是揭開了 DataLoader 的黑盒子，理解了它的運作原理。最後再重點整理一下 Dataloader 運作的流程：

* 當第一次調用 loader 的 `Load` 方法時，loader 內部會開啟一個 input channel 來接收查詢資料的 request，並開始等待 request 傳入。
* 每次調用 `Load` 方法時，都會建立一個 request（裏面包含 key 和一個接收查詢結果的 channel）並將其傳入 input channel。
* `Load` 方法返回一個匿名函數，函數內部如果沒有可以立即返回的結果，會試圖從 request channel 中取值，因此在尚未有值送入 channel 之前會暫時阻塞。
* loader 的 input channel 會等待一段時間（預設為 16ms），時間一到就關閉接收 request。（如果有設定接收 request 的最大數量，也有可能提前關閉）
* input channel 關閉後，就會將所有搜集到的 request keys 傳給自訂批次處理方法，取得所有資料（為 key - value 形式）。
* 資料載入完成後，將這些資料依照 key 推進對應的 request channel，接收到該 channel 回傳值的匿名函數就會停止阻塞，將結果緩存後，回傳給 client。

