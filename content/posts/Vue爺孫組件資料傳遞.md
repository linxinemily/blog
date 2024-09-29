---
layout: vue
title: Vue $attrs/$listeners 爺孫組件資料傳遞
date: 2019-06-23 22:38:02
tags:
- Vue
---

平常在實作 Vue 組件之間的資料傳遞大部分都是透過 `props` 及 `$emit`，或是直接經由 Vuex 進行狀態管理，而除了這兩種方法，還有另外一種做法是透過 `$attrs` 及 `$listenter`。
<!-- more -->
本身也是因為之前偶然在查資料的時候看到這兩個 API，但當時看過別人寫的文章後卻只是似懂非懂，頂多知道有這個東西但不知道如何使用或該用在什麼場合，一直到實作時才真正理解它的用法，但多看多讀總是好的，有一天或許派得上用場。

##  `$attrs` 
其實簡單說就是子組件可以透過`$attrs`取得父組件當中*除了 Props 以外*的所有資料。

乍看之下好像沒什麼，那不就直接用 Props 就好了嗎？
但當在多重組件嵌套的情況之下，就會顯得很有用。我們先定義出三個組件，他們分別是： a-component（爺組件/模板）、b-component（父組件）、c-component（孫組件）

我們可以在 a 使用 b：
``` html
// a-component
<template>
  <div>
    <b-component :msg="hello"/>
  </div>
</template>

```

接著在 b 中我們可以使用 `$attrs` 而不用透過 Props 拿到 `apiUrl`，並且將 `$attrs` 透過 `v-bind` 綁定在 c 上傳給 c 使用：
``` html
// b-component
<template>
  <div>
		<h1>{{ $attrs.data }}</h1>
		<c-component v-bind="$attrs"/> 
	</div>
</template>
```

然後同樣在 c 當中我們就能夠透過`$attrs`取得 a 的資料 。這樣一來的好處就是當我們有多個資料要從 a 傳給 b 跟 c 共用時，不用每一層都要聲明 Props，然後還要將每個 Props 寫在 component 的標籤裡面：
``` html
// c-component
<template>
  <div>
		<h1>{{ $attrs.data }}</h1>
	</div>
</template>
```

> 和 `$attrs` 相關的 API 還有 `inheritAttrs`，其值為布林值，預設為 `true`，也就是如果我們沒有特別將其設定成 `false`，子組件默認可以透過 `$attrs` 取得在父組件當中非透過 Props 繼承的資料。


## $listeners
應該不難猜想， `$attrs`  對應 Props ，`$listeners` 則對應 `$emit`。
簡單來說就是父組件可以透過 `$listeners` 取得所有子組件 `$emit` 打出來的事件。同樣以上面三個組件為例：

``` html
// c-component
<template>
  <div>
    <button @click="$emit('sayHi')"> click me!</button>
  </div>
</template>
```

在 b 當中同樣可以透過 `v-on="$listeners"`將 c 事件委任給 a：
``` html
// b-component
<template>
  <div>
    <c-component v-on="$listeners" />
  </div>
</template>
```

a 就可以直接處理 c 打出的`$emit`事件，而不用透過 b 再 emit 一次
``` html
// a-component
<template>
  <div>
    <b-component @sayHi="anEventHandler" />
  </div>
</template>
```

## 總結
所以當多重組件之間需要溝通傳遞，但不想用 Vuex 時，可以嘗試採用這兩種方法：
	* `$attrs`：孫組件取得爺組件的資料
	* `$listeners`：爺組件直接觸發孫組件的事件

