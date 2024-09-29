---
title: 解決使用 oapi-codegen 時報錯 unexpected reference depth 
date: 2023-03-17
tags:
- Golang
- OpenAPI
---

最近在研究 [oapi-codegen](https://github.com/deepmap/oapi-codegen) 這套基於 OpenAPI 3.0 自動生成 Go boilerplate 程式的工具，能幫助開發者省下很多撰寫實現 HTTP Server 端口、marshalling 和 unmarshalling 的重複程式碼的時間。當進行系統重構或使用像是文件先行(Documentation-Driven Development)的開發方法時也很適用。
在此紀錄一下餵入 OpenAPI 檔案時遇到的問題。


## 問題概述

使用以下 OpenAPI yaml 檔（擷取片段）

```yaml
schemas:
  Response:
    status:
      description: 狀態
      type: integer
      example: 200
    message:
      description: 訊息
      type: string
      example: success
    detail:
      description: 詳細訊息
      type: object
      example: {}

#...more

paths:
  /iot/stations:
    get:
      parameters:
        - name: uniqid
          in: query
          description: Uniqid
          required: false
          schema:
            type: string
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: object
                    properties:
                      items:
                        type: array
                     #...more
                  message:
                    $ref: '#/components/schemas/Response/message'
                  status:
                    $ref: '#/components/schemas/Response/status'
```

跑生成代碼的指令時

```bash
$ oapi-codegen -package main openapi.yaml > openapi.gen.go
```

會噴這個錯誤

```
error generating code: error creating operation definitions: error generating response definitions: error generating request body definition: error generating Go schema for property 'message': error turning reference (#/components/schemas/Response/message) into a Go type: unexpected reference depth: 5 for ref: #/components/schemas/Response/message local: true
```

看了一下[原始碼](https://github.com/deepmap/oapi-codegen/blob/master/pkg/codegen/utils.go#L322-L332)，發現他只能解析四層的 $ref path：

```go
// https://github.com/deepmap/oapi-codegen/blob/master/pkg/codegen/utils.go#L322-L332

// refPathToGoType returns the Go typename for refPath given its
func refPathToGoType(refPath string, local bool) (string, error) {
	if refPath[0] == '#' {
		pathParts := strings.Split(refPath, "/")
		depth := len(pathParts)
		if local {
			if depth != 4 {  // <---------here!
				return "", fmt.Errorf("unexpected reference depth: %d for ref: %s local: %t", depth, refPath, local)
			}
		} else if depth != 4 && depth != 2 {
			return "", fmt.Errorf("unexpected reference depth: %d for ref: %s local: %t", depth, refPath, local)
		}
```

也就是說，只有長這樣的 path 才能被順利解析：

```
#/components/responses/Baz 
```

而在上面的 yaml 當中的 path 已經到第五層  `#/components/schemas/Response/message` ，因此會報錯。


後來發現，這種寫法不是合法的 OpenAPI 格式的寫法（但用 [redoc](https://redocly.com/docs/redoc/quickstart/) 可以 build 出來），因為當跑 lint 的時候會報錯。

如果要改成合法的格式，首先 schema 的地方要在每個 Object 底下增加 `properties` 屬性：
```yaml
schemas:
  Response:
    properties:  # <-------- add this
      status:
      description: 狀態
      type: integer
      example: 200      
```
然後 $ref path 要改成 `#/components/schemas/Response/properties/message`
才能通過 linter 檢查。

但即使改成了合法格式，在用 oapi-codegen 生成代碼時還是會報錯。因為 path 總層數變成 6 層，依然無法被解析。
 

## 解決方法

### 1. 改成只使用四層的 $ref

像這樣：

```yaml
schemas:
  Resource:
    type: object
      properties:
        data:
          type: object
          properties:
            items:
              type: array
        #...more
        message:
        #...more
        status:
        #...more
paths:
  #...more
  responses:
    '200':
      description: OK
      content:
        application/json:
          schema:
          $ref: '#/components/schemas/Resource'
```

但這樣一來就變得很不彈性，在幾乎沒什麼 $ref 會引用到相同內容的情況下，還不如直接 inline，而且要改的地方有點太多了(懶)

### 2. 先把檔案修正成可以通過 OpenAPI 的 linter 檢查，然後再用工具 取代掉 $ref 的部分

想說反正只是為了能夠使用代碼生成工具，不想花太多精力大改原本的 OpenAPI 文件，於是最後採取了這個方法。

剛剛有稍微提到原本的格式跑 lint 時會有錯誤，需要進行修正，只要能夠成功跑過 linter 的檢查，就可以再用工具把 $ref 引用的地方全都取代成相對應的定義內容。[這篇問答](https://stackoverflow.com/questions/65549855/openapi-specification-yml-yaml-all-refs-replace-or-expand-to-its-definition)有提供兩種工具都能達成這個目的，在這邊我們選用了 [redockly-cli](https://redocly.com/docs/cli/)（沒有為什麼只是我們也有用 redoc）。

#### [run linter](https://redocly.com/docs/cli/commands/lint/)

首先修正上面剛剛提到的 properties 和 $ref 的部分後，就可以成功跑過 linter：

```bash
$ redocly lint openapi.yaml
```

![Untitled](images/oapi-codegen-2.png)

warning 的部分不用理他。

#### [bundle file and dereferenced](https://redocly.com/docs/cli/commands/bundle/)

然後再對檔案跑 bundle 指令：

```bash
$ redocly bundle --dereferenced openapi.yaml --output openapi_0317.yaml
```

![Untitled](images/oapi-codegen-1.png)

然後再一次跑 oapi-codegen 指令：

```bash
 $ oapi-codegen -package ports openapi_0317.yaml > openapi.gen.go
```

總算成功 gen 出檔案啦🎉

#### Note

在預設不帶任何 `-generate` 參數的情況下，該指令會幫我們產生所有包含 server, types, client, spec 等等相關程式碼在指定輸出的檔案裡，且預設使用的 server 是 Echo。（詳細說明參考[文件](https://github.com/deepmap/oapi-codegen#using-oapi-codegen)）

假設我們的需求是使用 gin ，並且只需要 server 和 types 部分的程式碼

> **`server`**：生成 server 接口以及相關的程式碼，包括將參數綁定到 struct、參數類型錯誤 throw Exception 的處理等等 <br>
**`types`**：定義在 OpenAPI components 當中的類型(struct)，包含 request params/body 和 response body
> 

並且分別放置於不同檔案中，可以這樣做：

先產生 gin server 的檔案

```bash
$ oapi-codegen -package ports -generate gin openapi_0317.yaml > openapi_api.gen.go
```

再產生 types 的檔案

```bash
$ oapi-codegen -package ports -generate types openapi_0317.yaml > openapi_types.gen.go
```