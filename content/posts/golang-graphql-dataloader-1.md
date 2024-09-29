---
title: "GraphQL Dataloader 在 Golang 當中的打開方法與原理解析（上）"
date: 2023-06-04T17:43:11+08:00
description: 介紹 GraphQL 的 Dataloader，以及在 Golang 當中的使用方法。
tags:
- GraphQL
- Golang
aliases:
- /posts/go-gqlgen-and-dataloader-1
---

## The Problem

使用 GraphQL 時，遇到以下的查詢：

```graphql
query {
  content(hotel_id: 1) {
    rooms {
      features {
        id
        icon_url
        text {
          zh_tw
        }
      }
    }
  }
}
```

在資料的關聯上，content 底下有許多 rooms，每個 room 底下都有多個 features。但在資料表結構上，room 有一個 `content_id` 欄位，以及一個 `features` 欄位，裏面儲存了所有 features 的 id。

而現在 Client 希望，當 query features 時如果夾帶了 `icon_url` 或 `text` 欄位，server 必須多回傳 feature 的這兩個欄位。也就是在 room.feature 的 resolver 的這兩個方法當中：

```go
func (r *roomFeatureResolver) IconURL(ctx context.Context, obj *model.RoomFeature) (*string, error) {
  // maybe query feature data from DB by obj.id ...?  
  return f.IconUrl, nil
}

func (r *roomFeatureResolver) Text(ctx context.Context, obj *model.RoomFeature) (*model.I18nString, error) {
  // maybe query feature data from DB by obj.id ...?  
  return f.Text, nil
}
```

都需要向 DB 取得 feature 資料。

但這麼一來，只要有 n 個 rooms，m 個 feature，就得 query `n * m * 2`(取得 icon_url 和 text 2 個欄位) 次，嗯..非常熟悉的 N+1 問題。
要解決這個問題，GraphQL 官方有提供一個方法，那就是使用 DataLoader。


## 什麼是 Dataloader？

DataLoader 是 GraphQL 官方提供的一個[套件](https://github.com/graphql/DataLoader)，主要用於解決資料載入時的效能問題。透過將多個請求收集到批次中，然後一次性獲取結果，從而減少重複的查詢或計算。
具體而言，DataLoader 有以下特點：

> *以下所提到的「請求」都是指在同一個 API Request 生命週期當中，對同一資料的讀取請求。*

* 批次處理：DataLoader 會將一段時間內或一定數量的請求進行批次處理載入數據，以減少資料庫的查詢次數和計算成本。
* 緩存：DataLoader 內部使用緩存機制，將已經載入的資料暫存在記憶體中。當下次請求需要相同的資料時，可以直接從緩存中返回，避免再次查詢資料庫。(但這些緩存數據只會存在於同一個 API Request 的生命週期當中，當 Request 結束後就會被清除)

* 避免重複載入：DataLoader 會確保同一個資料只被載入一次，即使同一時間有多個請求要求載入相同的資料，也只會執行一次載入操作。


但因為官方套件只有 nodejs 版本，其他語言的話官方是表示留給開發者們自己去實現...
> *"DataLoader is provided so that it may be useful not just to build GraphQL services for Node.js but also as a publicly available reference implementation of this concept in the hopes that it can be ported to other languages.“*

當然使用 Golang 的我們很幸運地還是可以繼續站在巨人的肩膀上，在 gqlgen（一個提供快速搭建 GraphQL Server 的套件）的文件當中，就直接端出了這個 Golang 版本的套件：[graph-gophers/DataLoader](https://github.com/graph-gophers/DataLoader)。所以下所討論的使用方法以及原始碼，都是會以這個套件的內容為主。

## 使用 Dataloader
那麽就來透過 DataLoader 實現我們的需求吧！

預期達到的效果是：**將 Query 當中所有 rooms.features 的 id 都搜集起來，一次向資料庫發送查詢，取得所有 feature 資料後，將其暫存在記憶體，當需要特定 id 的 feature 資料時便從中取出回傳。**

雖然使用方式在剛剛提到的 gqlgen [官方文件所提供的範例](https://gqlgen.com/reference/dataloaders/)當中都已經寫得很清楚，但下面還是會完整呈現，補充當中幾個方法跟參數的關係，以便在後續的原始碼閱讀中可以更容易理解：

```go
// my_repo/graph/storage/storage.go
package storage

type ctxKey string

const (
  loadersKey = ctxKey("dataloaders")
)

type FeatureReader struct {
  featureRepo model.FeatureRepository // 將向DB查詢的部分封裝到 Repository 當中
}

// 實際上會被 dataloader 所呼叫，用來一次取得所有 features 的方法
// 這邊的參數 keys 的來源是下方另一個同名的自定義方法當中，呼叫了 dataloader 的 Load 方法
// 而 Load 方法都會接收一個 key 參數
// 當搜集完成所有的 keys 後就會被傳進這邊來
func (u *FeatureReader) GetFeature(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
  featuresIDs := make([]int, len(keys))
  for ix, key := range keys {
    featureID, _ := strconv.Atoi(key.String())
    featuresIDs[ix] = featureID
  }
  featuresById, err := u.featureRepo.GetFeaturesByIDs(ctx, featuresIDs) // 實際向DB取資料的方法
  if err != nil {
    return nil
  }

  // 必須將資料的順序按照 keys 的順序排列
  output := make([]*dataloader.Result, len(keys))
  for index, featureKey := range keys {
    feature, ok := featuresById[featureKey.String()]
    if ok {
      output[index] = &dataloader.Result{Data: feature, Error: nil}
    } else {
      err := fmt.Errorf("feature not found %s", featureKey.String())
      output[index] = &dataloader.Result{Data: nil, Error: err}
    }
  }
  return output
}

type Loaders struct {
  FeatureLoader *dataloader.Loader
}

// 待會會在主程式裏面呼叫，實例化所有 loaders 以便傳入 middleware 當中
func NewLoaders(MySqlFeatureRepository model.FeatureRepository) *Loaders {
  featureReader := &FeatureReader{
    featureRepo: MySqlFeatureRepository,
  }
  loaders := &Loaders{
    //新增一個 Loader 並傳入自定義的批次處理方法
    FeatureLoader: dataloader.NewBatchedLoader(featureReader.GetFeature), 
  }
  return loaders
}

// 透過 middleware 將 loaders 注入到 context 裏面
// 以利其他地方可以透過同一個 ctx 物件取得 loader（一個 API Request 當中會有一個 ctx 物件）
func Middleware(loaders *Loaders, next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    nextCtx := context.WithValue(r.Context(), loadersKey, loaders)
    r = r.WithContext(nextCtx)
    next.ServeHTTP(w, r)
  })
}

// 將 loaders 從 context 當中取出的 helper method
func For(ctx context.Context) *Loaders {
  return ctx.Value(loadersKey).(*Loaders)
}

// 會在 resolver 當中被呼叫，只要傳入 feature id 就會回傳在 dataloader 當中準備好的對應資料
func GetFeature(ctx context.Context, featureID int) (*model.RoomFeatureWithData, error) {
  loaders := For(ctx)
  // 這邊所傳入 Load 的第二個參數，也就是上面自定義方法 FeatureReader.GetFeature 註解當中所提到
  // 每個傳進 Load 的第二個參數，也就是 key
  // 都會被搜集下來，直到所有 keys 搜集完畢後，就會觸發 FeatureReader.GetFeature 方法
  // 並將所有搜集的 keys 傳入該方法
  thunk := loaders.FeatureLoader.Load(ctx, dataloader.StringKey(fmt.Sprintf("%d", featureID))) // 留意 key 必須為字串類型
  result, err := thunk()
  if err != nil {
    return nil, err
  }
  return result.(*model.RoomFeatureWithData), nil
}

```

接著，到 resolver 當中，將取得資料的邏輯改成呼叫 `stroage.GetFeature` 方法：
```go
func (r *roomFeatureResolver) IconURL(ctx context.Context, obj *model.RoomFeature) (*string, error) {
  f, _ := storage.GetFeature(ctx, *obj.ID)
  return f.IconUrl, nil
}

func (r *roomFeatureResolver) Text(ctx context.Context, obj *model.RoomFeature) (*model.I18nString, error) {
  f, _ := storage.GetFeature(ctx, *obj.ID)
  return f.Text, nil
}
```

然後在主程式當中（或是定義路由的檔案），加上 middlerware：
```go
router.Use(func(next http.Handler) http.Handler {
    // 每次 request 都有自己的 dataloader
    return storage.Middleware(storage.NewLoaders(featureRepo), next) 
})
```

至此，不管往後總共有幾個 features，都只會執行一次的資料庫查詢，解決了 N+1 問題。
礙於篇幅有點太長，想說拆成上下兩篇比較好閱讀，所以分析 Dataloader 原始碼的部分，就留到[下一篇]({{< ref "/posts/golang-graphql-dataloader-2.md" >}})了～
