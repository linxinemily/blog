---
title: "Golang 的資料庫(整合)測試 Part 2 - 使用 Testcontainers"
date: 2023-05-17T15:43:34+08:00
description: 在 Golang 資料庫整合測試當中使用 Testcontainers 解決測試所需的外部依賴。介紹 Testcontainers 及其用法，提供使用範例參考。
tags:
 - Golang
 - Testing
---

## 前言

上一篇文章 [Golang 資料庫(整合)測試]({{< ref "/posts/golang-database-testing.md" >}}) 當中提到了在使用真實資料庫作為測試資料庫時，還可以使用 [testcontainers](https://golang.testcontainers.org/) 讓我們能夠在測試當中直接操作 Docker Container，去建立測試時需要的資料庫環境。這樣一來，也不用像一開始那樣，得另外手動配置一個專門拿來跑測試用的資料庫，將測試整合到 CI/CD 流程上時也比較方便。

<small> 然後應該是不會有 Part 3 啦。 </small>
## Testcontainers 簡介
[Testcontainers](https://golang.testcontainers.org/) 為多種語言提供套件（Golang、Java、Python 等等），開發人員可以透過該套件所提供的 API，在程式當中建立和清除基於容器的依賴項，以進行自動化整合測試。

所以就延續使用上篇文章當中的範例，把它改成使用 Testcontainers 的版本吧！

## 使用 Testcontainers

### Requirement
首先必須先確保環境當中有安裝 Docker，狀態為 running，且使用者要有權限可以執行 docker 指令。
不同的作業系統會有不同的建議版本和注意事項，可以參考[官方文件](https://www.testcontainers.org/supported_docker_environment/)。

> note: 原本在 local 跑，電腦是 Mac 又是 M1 晶片，遇到一堆環境問題快被搞瘋，後來索性放棄，改成在 linux 主機上面跑就都一切正常QQ

### 安裝 Testcontainers for Go 包

```
go get github.com/testcontainers/testcontainers-go

```

### 安裝 MySql Module 包

由於我們使用的 MariaDB 為 MySql 旁系血親，所以可以安裝官方提供的 [MySql Module 包](https://golang.testcontainers.org/modules/mysql/)，直接用包提供的 API 可以再多省下一些程式碼：
```
go get github.com/testcontainers/testcontainers-go/modules/mysql
```

### 調整程式碼

上回我們將重置資料庫的步驟放在 suite 的 setup 當中，那我們勢必得在這個動作之前先準備好資料庫，也就是要先把 MariaDB container 起起來：

``` go {linenos=table,hl_lines=["10-16","20-50"]}
import (
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
    testcontainermysql "github.com/testcontainers/testcontainers-go/modules/mysql"
)

// 添加了幾個屬性到 struct 裡面
type MysqlRepoTestSuite struct {
  suite.Suite
  dbUser     string
  dbPassword string
  dbName     string 
  dbHost     string
  dbPort     string
  mariadbC   *testcontainermysql.MySQLContainer
  ctx        context.Context
}

func (suite *MysqlRepoTestSuite) SetupSuite() {
  // ------開始啟動 container 的作業------
  suite.ctx = context.Background()
  mariadbC, err := testcontainermysql.RunContainer(
    suite.ctx,
    testcontainers.WithImage("mariadb:10.5"),
    testcontainermysql.WithPassword(suite.dbPassword),
    testcontainermysql.WithDatabase(suite.dbName),
    testcontainers.WithWaitStrategy(
        wait.ForLog("mysqld: ready for connections.").
            WithOccurrence(2).
            WithStartupTimeout(2*time.Minute),
    ),
  )

  if err != nil {
    log.Fatal("Failed to start MariaDB container: ", err)
  }

  suite.host, err = mariadbC.Host(suite.ctx) // 取得 container 的 host
  if err != nil {
    log.Fatal("Failed to get MariaDB container host: ", err)
  }
  // 將 container 3306 port 映射到主機上，並取得映射到主機上的 port
  // (等同下 docker run 指令時的 -P {主機port}:{container port} 操作)
  mappedPort, _ := mariadbC.MappedPort(suite.ctx, "3306/tcp")
  suite.container = mariadbC
  suite.port = strconv.Itoa(mappedPort.Int())

  if err != nil {
    log.Fatal("Failed to get MariaDB container port: ", err)
  }

  _, err = suite.freshDatabase()
  if err != nil {
    panic(err)
  }
}
```

傳入 `testcontainermysql.RunContainer()` 的參數除了 ctx 之外都是選填，分別解釋上面傳入各個參數的意義：
##### `testcontainers.WithImage("mariadb:10.5")`
指定使用 mariadb 10.5 版本的 docker image（不傳入該參數的話，預設會抓取 mysql:8 的 docker image）
#####  `testcontainermysql.WithPassword(suite.dbPassword)`
指定啟動 container 時傳入的環境變數 `MYSQL_PASSWORD`（不傳入的預設值是 `test`）
#####  `testcontainermysql.WithDatabase(suite.dbName)`
指定啟動 container 時傳入的環境變數 `MYSQL_DATABASE`，建立一個名為 `suite.dbName`(變數) 的 database（不傳入的預設值是 `test`）
#####  `testcontainers.WithWaitStrategy(wait.ForLog("mysqld: ready for connections.").WithOccurrence(2).WithStartupTimeout(2*time.Minute))`
由於下建立 container 的指令之後，需等待 container 完全啟動，才能保證後續的使用可以正常地連接上 container，因此這個參數是讓我們可以設定等待的策略。分別解釋鏈的各個方法意義：

* `wait.ForLog()`：程式會被 blocking 住，直到 container 當中特定的 log 出現後，才繼續往下執行
* `WithOccurrence()`：指定 log 必須出現的次數
* `WithStartupTimeout()`：指定等待的超時時間

用人話來總結這一行的操作： 等待 log 當中出現 `mysqld: ready for connections.`這段文字 `2` 次之後才繼續往下執行，如果 `2` 分鐘之內沒等到的話就跳 timeout 錯誤。


接著，就是要記得在 suite 的 teardown 當中，關閉 container 釋放資源：
``` go
func (suite *MysqlRepoTestSuite) TearDownSuite() {
  err := suite.mariadbC.Terminate(suite.ctx)
  if err != nil {
    panic(err)
  }
}
```

最後就可以來跑看看測試了：
```
go test
```

![database-testing-in-golang](images/testcontainer-go.png)
成功囉🎉

## 小結
其實使用的方式滿簡易的，概念上也不難，就是在 setup 當中最一開始多一個「啟動資料庫 Server 的 Docker Container」以及最後 teardown 時「關閉 container」的步驟而已，後面就可以繼續原本的流程(重置資料庫、開啟連線...等等)，但比較會遇到問題的部分可能會是在環境配置以及 API 的使用，因為官方文件的範例都沒有到很完整，使用時會需要摸索一陣子，因此才特別寫一篇筆記記錄。

最後附上完整的程式碼範例：
https://gist.github.com/linxinemily/276905e0145218538cb0be3a79a36153