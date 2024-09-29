---
title: "使用 vue-echarts 踩的坑"
date: 2019-06-11
tags:
- Vue
---

第一次使用 Echarts 這套圖形 library，在 vue 中使用 Echarts 可以直接裝 vue-echart，官方文件推薦兩個都裝：
```
$ npm install echarts vue-echarts
```
預料之中的踩了不少坑，在此紀錄一些重點。
<!-- more -->

## 安裝完後引入並註冊圖表 component
``` js
import Vue from 'vue'
import ECharts from 'vue-echarts'

import 'echarts/lib/chart/bar'
import 'echarts/lib/component/tooltip'

import 'echarts-gl' //等等會使用 grapic 設定圖形文字，故需載入此 module

Vue.component('v-chart', ECharts) // 註冊為 global component

```

## 基礎使用
在 template 中可以直接使用圖表 component，其接收 `options` 為 props，裡面的`series`屬性為設置特定類型圖表的屬性，在此使用專案中的圓餅圖當範例：

``` html
 <v-chart :options="options" />
```

``` js
data() {
    return {
        options: {
            series: [
                {
                    type: 'pie',  // 必須
                    name: 'test', // 必須
                    radius: ['50%', '70%'],
                    label: {
                    normal: {
                        show: false
                    },
                    emphasis: {
                        show: true,
                        formatter: '{b} {@percentage}%'
                    }
                    },
                    data: [
                    { name: '韓國', value: 70 },
                    { name: '日本', value: 10 },
                    { name: '台灣', value: 10 },
                    { name: '越南', value: 6 },
                    { name: '中國', value: 3 },
                    { name: '其他', value: 1 }
                    ]
                }
            ]
        }
    }
}
```


## 圓餅圖中間加入文字
> 注意：**一定要裝 `echarts-gl` 模組**，否則會無法顯示。
可以直接利用官方提供的屬性`grapic`為圖表自行添加圖形元素，並新增特定文字（可以新增的圖形元素類型還有圖片、圖形等，可以參考[文件](https://echarts.baidu.com/option.html#graphic)），格式如下：
``` js
graphic: [
          {
            type: 'text',
            left: 'center',
            top: 'center',
            style: {
              text: '學生國籍比例',
              textAlign: 'center',
              fill: '#666',
              font: '20px "STHeiti", sans-serif'
            }
          }
        ]
```

## 圖例說明（legend）需要添加 value 資料
echarts 提供的 formater callback function 只有給我們 `name` 這個參數，所以要同時拿到 `value` 的話，會需要自己從圖表實例裡面找。
> 注意：**必須使用箭頭函式**才能拿到這個 component 的實例，進而取得 `options.series[0].data` 得到我們要的 `value`
``` js
legend: {
          orient: 'verticle',
          left: 'left',
          bottom: 'middle',
          textStyle: {
            fontSize: 16
          },
          formatter: name => {
            const data = this.options.series[0].data
            let targetValue
            data.map(item => {
              if (item.name === name) {
                targetValue = item.value
              }
            })
            return name + ' ' + targetValue + '%'
          }
        },
```

## 圖表的響應
由於一開始設定圖例的排列方式為垂直排列、靠左並置中，但在行動裝置上這樣會破版，所以首先第一步先設定讓整個 Canvas 可以自適應容器大小縮放，實現方法其實只要修改 CSS 即可：

``` css
.echarts {
  width: 100%;
  height: 400px; 
}
```

接著就會面臨到圖例跟圖表本體重疊的問題，所以我們需要偵測視窗寬度大小，隨著裝置寬度不同改變圖例排列的方式。
#### 1. 首先先在 component 上綁定 `ref`
``` html
<v-chart ref="myChart" :options="options" />
```

#### 2. 在 mounted 當中監聽 window 的 load event，當頁面載入時計算當前螢幕寬度大小，再透過 `ref` 屬性訪問該圖表實例並修改其 `options.legend` 屬性（和圖例定位相關的屬性）

``` js
window.addEventListener('load', () => {
      this.windowWidth = window.screen.width
      let orient, left, bottom

      if (this.windowWidth < 480) {
        orient = 'horizontal'
        left = 'center'
        bottom = 'bottom'
      } else {
        orient = 'vertical'
        left = 'left'
        bottom = 'middle'
      }
      if (this.$refs.myChart) {
        this.$refs.myChart.options.legend.orient = orient
        this.$refs.myChart.options.legend.left = left
        this.$refs.myChart.options.legend.bottom = bottom
      }
    })
```
> 注意：此綁定方法只有在頁面第一次渲染時才會有效，如果要實現 RWD 要再另外監聽 resize 事件。

## 結語
#### 1. 如果要用到較多客製化功能要用 Echarts，否則才用 v-charts
在加入圓餅中間文字那裡卡滿久，原因是一開始裝的是 v-charts（算是輕量簡化版的 Echarts），推測目前還沒有支援到可以使用圖形文字（各種方式加入 `graphic` 屬性都未起作用），雖然也可能只是尚未找到解決方案（翻文件快翻到發瘋）。

#### 2. 在 Vue 當中取得圖表實例的方法
在 `options` 內部若用到其提供的回調函式，必須使用箭頭函式，拿到的 `this`才會是我們要的圖表實例物件（否則會是 undefined）。但在其他地方（Vue 的環境中）則要透過訪問 DOM 元素才能得到圖表實例物件。
目前的方法是這樣，但其實官方好像有提供其他可以訪問 `options`的 API（如 `computedOptions`，可參考[文件](https://github.com/ecomfe/vue-echarts#computed)） ，但我目前是都沒有成功過 QwQ，可能還需另外研究。