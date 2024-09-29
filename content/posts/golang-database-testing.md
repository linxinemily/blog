---
title: Golang 的資料庫(整合)測試
description: 本文記錄了探索在 Golang 當中撰寫資料庫整合測試的過程及方法。包含：進行資料庫測試時，該使用真實的測試資料庫或是測試替身(Test Double)？該如何重置資料庫狀態，才能避免不同測試方法之間的資料相互影響？

date: 2023-04-23
tags:
- Golang
- Testing
---
![database-testing-in-golang](images/golang-database-testing-1.png)

## TL;DR

本文記錄了探索在 Golang 當中撰寫資料庫整合測試的過程及方法。包含幾個重點的討論：
1. 進行資料庫測試時，該使用真實的測試資料庫或是測試替身(Test Double)？
2. 使用測試資料庫的話，該如何重置資料庫狀態，才能避免不同測試方法之間的資料相互影響？
3. 最後提供讓不同測試方法之間資料不互相影響，並能平行跑測試的實作方法

## 前言

有經驗的工程師都知道寫測試對於軟體開發的重要性，而當中最基本的就是單元測試。狹義或者說比較嚴格定義的單元測試，如果以 Clean Architechture 的觀點來看，通常是針對 Domain Layer 以及 Application(Use Case) Layer ，也就是不涉及外部服務或套件/架等等的測試。但對於一些小型專案來說，Domain 邏輯通常不多，大部分都是對資料庫的 CRUD 操作，又或者使用的框架原本就把 ORM 跟 Model 綁死在一起，框架提供的測試集成 API 也是直接預設好對真實資料庫的連接，整個專案可能幾乎沒有幾個狹義的單元測試了。

不過硬是要畫分出單元測試或者整合測試其實沒什麼意義，這邊只是想要稍微點出一道現實，在和資料庫高度掛鉤或者所謂資料密集型的服務底下，資料庫測試勢必是極重要的一環。

然而在 Golang 當中，並沒有像是 Spring Boot or Laravel 這種包山包海的框架（似乎也不太可能出現，畢竟違反了 Golang 的設計哲學），測試的部分，雖然標準庫有提供一個[測試套件](https://pkg.go.dev/testing)，但僅是對於單元測試以及性能測試的輔助，至於資料庫測試？得自己想辦法。

> 還是可以找得到對於資料庫測試的輔助套件，但因爲泛用性沒有到非常廣，且實現資料庫測試的方法其實也並不難，所以就沒有特別去使用，這邊也不多加介紹。
> 

## 情境概述

參考 Clean Architecture 的架構，專案當中有劃分 Domain、Application、Adapter 以及 Port Layer。而在 Domain Layer 當中我們定義了 `UniqidRepository` 介面：

```go
//domain layer
type UniqidRepository interface {
  GetUniqidsWithDevicesFilterByUniqId(ctx context.Context, uniqid string) ([]*Uniqid, error)
}
```

這個介面會由 Adapter Layer 當中不同的 Repository 去實作，在這邊我們定義一個 `PostgresUniqidRepository` ，實現 `UniqidRepository` 介面，並且內嵌一個資料庫連線物件(使用 了 gorm 提供的 struct)：

```go
type PostgresUniqidRepository struct {
  db *gorm.DB
}

func (u *PostgresUniqidRepository) GetUniqidsWithDevicesFilterByUniqId(ctx context.Context, uniqid string) ([]*domain.Uniqid, error) {
  ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
  defer cancel()

  // to retrieve data from DB by GORM...

  return uniqids, nil
}
```

## 資料庫測試的選擇：測試替身 VS 真實資料庫

為了要撰寫 `PostgresUniqidRepository` 的測試，我們會需要依賴一個 `db` 物件，那這時候應該連接一個真實的測試資料庫，還是使用測試替身(Test Double)呢？先來看看這兩者各自的優缺點：

### 使用測試替身

測試替身是一個模擬資料庫行為的物件或程式，用來替代真實的資料庫連接。方法就是將原本實作連接真實資料庫的物件藉由依賴注入抽換成測試替身（之所以可以抽換，通常是因為有定義了一個介面，不論是連接真實資料庫的物件或是測試替身，都會去實現這個介面）透過測試替身去模擬資料庫的行為和回傳值。

> 在 Golang 當中，如果是使用 SQL 類型的資料庫，官方標準庫就有定義了 `sql.DB` 介面，也有相關的 library 像是 [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) 當中就提供了實現該介面方法的測試替身，讓開發者能在測試時能夠檢查所執行的 SQL 語句是否如預期，並模擬資料庫的回傳值。如果是使用其他類型的資料庫，例如 AWS 的 DynamoDB，其提供的 client sdk 當中也有包括 [dynamodb 介面](https://docs.aws.amazon.com/sdk-for-go/api/service/dynamodb/dynamodbiface/)，也是相同的概念。

- 優點：
    - 提供了較高的測試控制，容易模擬錯誤情境(ex: 模擬資料庫連接失敗)。
    - 由於不需要真實的資料庫連接，測試速度可能會更快。
- 缺點：
    - 可能無法完全模擬真實資料庫的行為，因此測試結果可能與實際運行時的行為不一致。
    - 需要額外的開發和維護工作，包括模擬資料庫的行為和維護測試替身的狀態，可能增加開發和維護成本。

> 剛剛有提到說包山包海的框架中，例如 Laravel，官方是沒有直接提供使用測試替身資料庫的選項，不過有 Memory 資料庫(sqlite)，這又是另一種資料庫測試的選擇，但這邊不特別介紹，暫且把他歸類在使用真實資料庫的範疇，因為依然是依賴外部服務。


### 使用真實資料庫

一個實際運行在主機或容器上的資料庫，通常專門 for 測試用。

> 有網友補充，還可以使用像是 [testcontainers-go](https://github.com/testcontainers/testcontainers-go) 這種工具，讓開發者可以透過呼叫其 API 來管理和使用 Docker 容器，進行測試和集成測試。能夠在測試當中更方便地管理資料庫依賴。介紹與用法可參考[下一篇筆記]({{< ref "/posts/golang-database-testing-with-testcontainer.md" >}})。
 
- 優點：
    - 提供更接近實際運行環境的測試，測試結果更加真實可靠。
    - 可以測試到真實資料庫的特定行為、效能等方面的問題，更全面地驗證系統的品質。
- 缺點：
    - 測試結果可能受到真實資料庫狀態和環境的影響，較難以控制測試情境。
    - 可能會因為資料庫中的變動而導致測試結果不穩定，像是不同測試案例之間的資料相互影響，需另外透過適當地調整測試順序、編寫清理腳本或確保不同測試案例使用資料的獨立性等方法來避免。

所以到底要用使用測試替身還是真實資料庫呢？

回到一開始範例的情境，有一個 Repository 專門用來連接 Postgres 資料庫，同時又用了 gorm 這個 ORM 套件包裝真實的 SQL 語法。如果要使用測試替身的話，可以借助 [go-sqlmock](https://github.com/DATA-DOG/go-sqlmock) 這個套件。但仔細想想，以 `GetUniqidsWithDevicesFilterByUniqId` 這個方法來看，僅有的邏輯就是取得所有/被過濾的 Uniqid 資料而已，如果又用了測試替身，那麽這個測試就變成只剩下檢查 SQL 語法有沒有組對。這樣算是一個有用的測試嗎？

以這個針對 Repository 測試的情境而言，**是否有確實從資料庫中取出正確的資料**，才是我們真正想要驗證的事情。因此，我認為使用真實資料庫會是比較適合的做法。

> 查詢資料的過程中，有看到一篇由 Github 工程師所撰寫的[部落格文章](https://markphelps.me/posts/writing-tests-for-your-database-code-in-go/)，當中也有提到，”**You need to actually test your SQL code.”**
> 

## 資料庫整合測試：重置資料庫狀態的方式

如果要進行連接實際資料庫的整合測試，為了避免每個測試彼此之間的資料互相影響，通常會在每次跑測試之前先重置資料庫狀態。重置資料庫狀態的方式有很多種，參考了手邊最常使用的 Laravel 框架，當中有提供許多協助在測試時重置資料庫狀態的 Trait，都是平常開箱即使用的方法：

- [DatabaseMigration](https://laravel.com/api/9.x/Illuminate/Foundation/Testing/DatabaseMigrations.html)：在每個測試方法執行之前，先刪除資料庫中的所有資料表，並重新 migrate，等該測試方法跑完後，再 rollback 所有 migrations。
- [DatabaseTransactions](https://laravel.com/api/9.x/Illuminate/Foundation/Testing/DatabaseTransactions.html)：在每個測試方法執行之前，開始一個 transaction，等該測試方法跑完後，再將 transaction rollback，回復到資料庫原本的狀態。
- [RefreshDatabase](https://laravel.com/api/9.x/Illuminate/Foundation/Testing/RefreshDatabase.html)：在每個測試方法執行之前，(非使用內存資料庫的狀況下)會先判斷資料庫的狀態是否已經 migrate 過，如果還沒就刪除資料庫中的所有資料表、重新 migrate，並將 migrate 狀態標示為已經 migrate 過 ，接著繼續執行和  DatabaseTransaction 相同的行為。

從這些 Laravel 所提供的 Trait 當中，大概可以整理出兩個方案：

1. **在每次跑測試方法之前，先跑 migration，等測試方法執行完畢後，再 rollback migration。**
好處就是簡單方便，但由於需要在每個測試方法當中都跑一遍 migration 再 rollback，連接真實資料庫的情況下，速度會比較慢。（所以大多是使用內存資料庫如 sqlite 時才會使用這個方法）
2. **在每次跑測試方法之前先開啟一個資料庫交易(begin transaction)，等測試方法執行完畢後，再回滾交易(rollback transaction)。**
不需要跑每個測試方法時都真的把整個資料庫重置，而是利用交易的特性，將每個測試當中對資料庫進行的操作都視為原子性的操作，執行速度較快。但需另外注意的是，不同測試方法如果都對同一筆或同一區間的資料做操作，可能會產生非預期的資料，進而影響到測試的結果。所以使用這個方法時，通常會在每個測試當中創建該測試需要用到的資料，而非和其他測試共享資料，如此一來，可以降低測試之間相互影響的風險，確保每個測試的獨立性。

但由於要使用真實資料庫來測試，以我們的情境來看，使用第二種方式會是比較適當的選擇。

> 其實重置資料庫狀態的方式不只這幾種，但 Laravel Trait 所提供的方式已經能滿足我們基本的需求，就不再額外探討。


## 實作資料庫整合測試

總算進入到實作測試的部分了！

在這邊會使用到幾個第三方套件來輔助測試以及資料庫的連接：

- [testify/suite](https://github.com/stretchr/testify#suite-package)：testify 測試框架當中的 suite package 方便用來對整體/分組/單獨的測試方法創建 setup 以及 teardown 方法。只要內嵌`suite.Suite` struct，就可以藉由複寫以下方法，來實現為不同的測試行為加入前/後置處理：
    - `SetupSuite()` ：執行 suite 當中**所有**測試方法之前的前置處理
    - `TearDownSuite()` ：執行 suite 當中**所有**測試方法之後的後置處理
    - `SetupTest()`：執行 suite 當中**每個**測試方法之前的前置處理
    - `TearDownTest()` ：執行 suite 當中**每個**測試方法之後的後置處理

![suite 有點類似測試方法的集合(組)，`SetupSuite()` / `TearDownSuite()` 為整組測試前/後的處理、 `SetupTest()` / `TearDownTest()` 為組內每個單獨測試/前後的處理](images/golang-database-testing.png)

- [go-migrate](https://github.com/golang-migrate/migrate)：使用 CLI 或是在程式當中呼叫特定方法執行定義好的 migration 檔案。
- [gorm](https://github.com/go-gorm/gorm)：Golang 當中著名的 ORM 框架。由於 Repository 當中使用了該套件存取資料庫，所以執行 Repository 方法時需要使用到。

我們可以在 suite 的 setup 當中，先重置資料庫並開啟一個連線。然後將每個測試方法都用一個 transaction 包住（在開始前先 begin TX、結束後 rollback TX），最後在 suite teardown 時關閉資料庫連線：

```go
type PostgresRepoTestSuite struct {
  suite.Suite
  db *gorm.DB
  tx *gorm.DB
}

func (suite *PostgresRepoTestSuite) SetupSuite() {
  // 重置資料庫，先把資料庫 rollback 或整個 reset 後再跑 migrations
  // 這邊的 migrations 應僅包含 DDL，並不會塞入任何一筆資料
  _, err := freshDatabase() 
  if err != nil {
    panic(err)
  }
  // 開啟一個資料庫連線並暫存，以便讓 suite 底下每個測試方法共用
  suite.db = openDatabaseConnection()
}

func (suite *PostgresRepoTestSuite) TearDownSuite() {
  sqlDB, _ := suite.db.DB()
  sqlDB.Close() // suite 底下所有測試方法跑完後，關閉資料庫連線
}

func (suite *PostgresRepoTestSuite) SetupTest() {
  suite.tx = suite.db.Begin()  // 使用暫存的 DB 連線物件，開啟一個 transaction
}

func (suite *PostgresRepoTestSuite) TearDownTest() {
  suite.tx.Rollback() // 回滾 transaction
}

func TestPostgresRepoTestSuite(t *testing.T) {
  suite.Run(t, new(PostgresRepoTestSuite)) // 執行 suite 底下所有測試方法
}

// 實際的測試方法，必須為 suite 的 receiver
func (suite *PostgresRepoTestSuite) TestGetUniqidsWithDevicesFilterByUniqId() {
  // 主要的測試邏輯
  // ...
}
```

在 `PostgresRepoTestSuite` 中， `db` 和 `tx` 變數是由 suite 底下的每個測試方法所共用的。在單一執行緒下執行測試時，這並不會造成任何問題。但如果要平行執行測試，這些方法在同一時間可能會競爭同一個變數，導致資源競爭的情況，進而影響測試結果的正確性。此外，這些方法執行時都是使用同一個資料庫連線，可能也會產生意外的錯誤。

於是我們可以改成在每個測試方法當中，都開啟專屬的資料庫連線，並創建獨立的變數保存連線實體，順便模擬平行跑測試時的場景：

```go
type PostgresRepoTestSuite struct {
  suite.Suite
}

func (suite *PostgresRepoTestSuite) SetupSuite() {
  _, err := freshDatabase()
  if err != nil {
    panic(err)
  }
}

// 覆寫了原本內嵌的 suite.Suite 預設 Run 方法
// 以便為每個方法創建獨立的資料庫連線物件
// 同時使用標準庫 *testing.T 提供的 Run 方法 wrap 起來，待會就能夠平行地執行這些子方法
func (suite *PostgresRepoTestSuite) Run(t *testing.T, method string, fn func(t *testing.T, tx *gorm.DB)) {
  t.Run(method, func(t *testing.T) {
    var db *gorm.DB
    var tx *gorm.DB

    defer func() {
      sqlDB, _ := db.DB()
      sqlDB.Close()
    }()
    defer func() {
      tx.Rollback()
    }()

    db = openDatabaseConnection()
    tx = db.Begin()

    fn(t, tx)
  })
}

func TestPostgresRepoTestSuite(t *testing.T) {
  t.Parallel() // 使用 goroutine 同時運行所有測試方法
  suite.Run(t, new(PostgresRepoTestSuite))
}

func (suite *PostgresRepoTestSuite) TestGetUniqidsWithDevicesFilterByUniqId() {
  // 呼叫剛剛覆寫的 Run 方法，將原本測試邏輯包裝成 function 做為參數傳入
  suite.Run(suite.T(), "TestGetUniqidsWithDevicesFilterByUniqId", func(t *testing.T, tx *gorm.DB) {
    // 主要的測試邏輯
    // ...
  })
}

```

但要注意的是，即使每個測試方法都使用各自的資料庫連接，如果兩個測試方法都嘗試同時對同一條資料進行存取，或者對某個範圍進行存取，也有可能存在資料庫層面的事務競爭。不過如果每個測試方法都只是操作自己創建的資料，迸發測試應該是安全的。

## 小結
重點整理如下：

* 如果要測試的 function 都和資料庫互動為主，通常會建議使用真實資料庫測試而非資料庫替身。
* 重置資料庫狀態可以藉由在每個測試方法執行前/後分別 run/rollback migrations 或開啟/回滾一個資料庫交易。使用交易通常會更省資源，但必須確保每個測試方法當中所使用資料的獨立性。
* 實作測試方法：
  1. 在 suite 的 setup 當中，重置整個資料庫後再跑 migration（僅包含 DDL，此時資料庫沒有任何一筆資料）
  2. 在 suite 底下的每個測試法方當中**按照順序**執行：
     1. 開啟獨立的 DB 連線
     2. 開始交易
     3. 運行測試邏輯（當中才建立所需的資料）
     4. 回滾交易
     5. 關閉 DB 連線
  
  如此一來，DB 連線和資料都不會互相干擾，也可以平行跑測試。
