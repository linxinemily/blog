---
title: "Promise.all() 與 async/await"
date: 2019-06-30 14:48:12
tags:
- Javascript
---

本來一直都用迴圈去處理同時發多個 request（很好懂但有點難處理 Error），這次親自來試試 Promise.all() 總算體驗到平行處理非同步的威力（？）於是紀錄一下心得以及和 await + axios 連用時的一些眉角。
<!-- more -->
## Promise.all() 用法
Promise.all (`iterable`) 接受一個 iterable 物件（通常是陣列）作為參數，當裡面每個元素都是 Promise 物件時(也可以不是，待會會提到），它就會平行去處理這些 Promise。由於**其本身會回傳一個 Promise 物件**，所以我們同樣可以用 then 或 await 去接他的 resolve/reject。
當所有的 Promise 都被 resolve 後才會回傳 resolve，其值會是全部的 Promise 處理完後所回傳的 resolve 值所組成的陣列。但若其中有一個 Promise 被 reject 就會直接拋出該 Promise 所回傳的錯誤。

``` js
const array1 = [1, 4, 9, 16];
const promiseArr = array1.map(x => Promise.resolve(x));
console.log(promiseArr) // [Promise, Promise, Promise, Promise]

// 用 then 接結果
Promise.all(promiseArr).then(vals => {
	console.log(vals) // [1, 4, 9, 16]
})
// 用 await 接結果
const vals = await Promise.all(promiseArr)
console.log(vals) // [1, 4, 9, 16]
```

> 提醒： `await` 後面接的一定會是一個  Promise （或 async function），他會幫我們把 Promise 處理完後直接吐給我們 resovle 或 reject 的值。

而當陣列裡面有非 Promise 物件的元素存在時，Promise.all() 仍會輸出該元素的值：

```js
let a = 1
let b = Promise.resolve(2)
let result = await Promise.all([a, b])
console.log(result) // [1, 2]
```
 
## Promise.all() + axios 

我們可以先透過 `map()` 處理原始陣列（`map()`會回傳新的陣列），把原始陣列改成一組 Promise 陣列。以下面程式碼為例， products 為一組商品物件的陣列，在此即利用 `map ()` 遍歷每個商品物件，在 `map()` 的 callback function 裡直接呼叫 axios 將商品資訊帶入 request body 發送請求，由於 axios 本身就會回傳 Promise，所以我們就能成功拿到一組 Promise 陣列並直接塞進 Promise.all 裡面囉！

``` js
let data = await Promise.all(this.products.map((product) => {
  return this.$axios.post('http://localhost:8000/api/products', {
    name: product.name,
    price: product.price,
    order_id: myOrder_id
  })
}))
console.log(data)
```

所以我們會得到一個陣列長這樣：


這時發現它長得好像不是我們要的 response data，那是因為 axios 的 response （resolve value）並非直接回 server 給我們的資料 ，而裡面那層 data 才是我們要的資料：
``` js
{
  data: {}, //這才是 server 回給我們的資料
  status: 200,
  statusText: 'OK',
  headers: {},
  config: {},
  request: {}
}

```