---
title: Mock Dynamodb Client(use aws SDK for golang) with testify/mock
date: 2022-08-04
tags:
- Golang
- Testing
---


測試時若涉及第三方服務，往往會需要運用到 Mock （測試替身）的概念，將外部模組代換成一個假的物件，並模擬其行為與回傳值，讓我們能專注於業務邏輯的單元測試。

當前的情境為，在 Repository 裡使用 [aws SDK for go](https://github.com/aws/aws-sdk-go) 當中的 dynamodb client 來建立與 dynamodb 的連線以及後續相關對資料庫的操作。但在進行測試時，為了避免每次跑測試都要真的戳資料庫（因在單元測試中，我們只想要確定與 dynamodb client 的互動有**確實呼叫預期的方法、傳入相應的參數**，而不在乎資料庫到底有沒有寫入資料這種事），會需要將這個 client 替換成 Mock 物件。

而 [aws SDK for go](https://github.com/aws/aws-sdk-go) 就提供了一個方便開發者進行 mocking 的 interface：[dynamodb interface](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbiface/) 

我們可以直接使用 [dynamodb interface](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbiface/) 作為依賴的類型，也就是說如果要放一個 dynamodb client 在 struct 裡面，原本可能會直接使用 `*dynamodb.DynamoDB` 類型：

```go
type dynamodbUserNotificationRepository struct {
  Client *dynamodb.DynamoDB
}

// 實例化時注入 dynamodb client 依賴
func NewDynamondbUserNotificationRepository(Client *dynamodb.DynamoDB) domain.UserNotificationRepository {
  return &dynamodbUserNotificationRepository{Client}
}
```

但為了之後方便在測試時能輕鬆注入 Mock 物件，可以替換成 `dynamodbiface.DynamoDBAPI` ：

```go
type dynamodbUserNotificationRepository struct {
  Client dynamodbiface.DynamoDBAPI  // instead of *dynamodb.DynamoDB
}

func NewDynamondbUserNotificationRepository(Client dynamodbiface.DynamoDBAPI) domain.UserNotificationRepository {
  return &dynamodbUserNotificationRepository{Client}
}
```

繼續完整這個範例，假設 `dynamodbUserNotificationRepository` 有一個 `Store` 方法：

```go
func (d *dynamodbUserNotificationRepository) Store(ctx context.Context, user_notification_input *domain.UserNotificationRequestInputStore, batch string) (user_notification *domain.UserNotification, err error) {
  // 裡面會呼叫到 d.Client.PutItem 方法
}

```

要測試上面這個 `Store` 方法，需要將 `dynamodbUserNotificationRepository` 裡面的 `Client` 取代成 Mock 物件，所以我們先按照官方文件範例定義一個 Mock client struct：

```go
type MockDynamoDBClient struct {
  dynamodbiface.DynamoDBAPI
}
```

接著複寫 `PutItem` 方法，回傳 dummy 資料：
```go
func (m *mockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput,error) {
  return &dynamodb.PutItemOutput{}, nil
}
```
然後就可以在測試中 new 一個 `mockDynamoDBClient` 傳入 `NewDynamondbUserNotificationRepository`：
```go
func TestStore(t *testing.T) {
  mockSvc := &mockDynamoDBClient{}
  repo := repository.NewDynamondbUserNotificationRepository(mockSvc)
  // do something...
}
```

感覺就很開心地寫完測試了。但如果同時有另外一個方法也會呼叫 `PutItem` ，而在為這個方法寫測試時，`PutItem` 的回傳值必須不同，該怎麼辦？首先想到的可能是再另外寫個 Mock client 2，但有 n 個難道要寫 n 次嗎？

這就是 testify/mock 出場的時候了，使用它能讓我們在個別測試裡定義 Mock 物件會被呼叫的方法名及其回傳值。

將 `MockDynamoDBClient` 修改為：

```go
type MockDynamoDBClient struct {
  dynamodbiface.DynamoDBAPI // 1. 
  mock.Mock // (多加上的部分) 2. 
}
```

這裡同時使用到兩個 package 提供的 interface 以及 struct

1. [aws/aws-sdk-go dynamodbiface.DynamoDBAPI](https://github.com/aws/aws-sdk-go/blob/v1.44.69/service/dynamodb/dynamodbiface/interface.go#L62-L286): 
依照[官方文件範例](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbiface/)，將 `dynamodbiface.DynamoDBAPI` interface 用匿名的方式嵌進 Mock client struct 裡面，使得該 struct 等同”繼承”了這些 interface 當中的方法，因此可以被視為一個有實現該 interface 的 struct。
2. [stretchr/testify mock.Mock](https://github.com/stretchr/testify/blob/master/mock/mock.go#L266-L283): 
依照官方範例，在定義 Mock client struct 當中嵌入 `mock.Mock`（同樣會有”繼承“ `mock.Mock` 裏的方法的效果）。

> 兩者都利用到了 golang struct embed 的特性，關於 golang 的 struct with embedded anonymous interface 有空會再寫另一篇筆記來解釋。
Ref：[https://stackoverflow.com/a/24546029/10943670](https://stackoverflow.com/a/24546029/10943670)
> 

接著定義 mock client 待會在測試中會被呼叫到的方法：

```go
func (m *MockDynamoDBClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput,error) {
  arg := m.Called(input) // 1.
  return arg.Get(0).(*dynamodb.PutItemOutput), arg.Error(1) // 2.
}
```

1. **取得參數**: 
這裏的 `Called` 方法會取得 `PutItem` 方法被呼叫之後應該要回傳的參數，待會會在寫測試時實際定義，總之這邊就是預期會得到一組在測試時定義的回傳參數。
2. **類型檢查**: 
檢查得到的參數是否符合類型，是的話才會真的將其當成 `PutItem` 方法的回傳值傳出去。 `arg.Get(0).(*dynamodb.PutItemOutput)` 表示得到的第一個參數必須是 `*dynamodb.PutItemOutput` 類型，`arg.Error(1)` 表示得到的第二個參數必須是 `Error` 類型，若不符合類型會引發 panic。

> 官方文件：[https://github.com/stretchr/testify/blob/master/mock/doc.go](https://github.com/stretchr/testify/blob/master/mock/doc.go)
> 

最後來跑一次加入了 `mock.Mock` 之後的整個測試流程吧：

```go
func TestStore(t *testing.T) {

  mockSvc := &mockDynamoDBClient{} // 1.

  mockSvc.On("PutItem", mock.Anything).Return(&dynamodb.PutItemOutput{}, nil).Once() // 2.

  repo := repository.NewDynamondbUserNotificationRepository(mockSvc) // 3.

  // other arrangements...

  // action
  ret, err := repo.Store(context.TODO(), mockedInput, batch) // 4.
  
  // assertions...
}
```

1. 首先實例化一個 Mock client
2. 重點在於這個部分，使用了 `mock.Mock` 當中的方法，分別解釋各個方法作用：
    - `On("PutItem", mock.Anything)` ：斷言呼叫該物件的 `PutItem` 方法時會收到一個任意參數 `mock.Anything`
    - `Return(&dynamodb.PutItemOutput{}, nil)` ：也就是上面提到的 `Called` 方法被呼叫後的回傳值
    - `Once()` ：斷言該方法應該要被呼叫一次
3. 再來將 Mock client 物件注入 `NewDynamondbUserNotificationRepository`
4. 呼叫 `repo.Store(context.TODO(), mockedInput, batch)` 時，裡面將會呼叫  Mock client 的 `PutItem` 方法

以上就是結合 [aws SDK for go](https://github.com/aws/aws-sdk-go) 所提供的 [dynamodb interface](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbiface/) 搭配 testify/mock 進行單元測試的方法～