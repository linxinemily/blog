---
title: "[leetcode 筆記] 11. Container With Most Water"
date: 2022-05-01
tags:
- Algorithm
---

[題目連結](https://leetcode.com/problems/container-with-most-water/)

## 題意簡述

給予一個正整數陣列，每個數字代表垂直線的高度，每條線之間距離為 1（對應元素索引位置）。用 xy 軸來表示會長這樣：

![題目的圖](https://cdn-images-1.medium.com/max/2000/0*7LVF1wZsVW5przXd)*題目的圖*

要求在這個圖形下，所能夠取得的最大面積。以上面這個題目給的例子，7（底）*7（高）= 49 為最大面積。（將每個元素當成柱子的高度，要求能夠獲得最大的儲水量）

## 解題思路

這題可以使用雙指針的技巧，舉例輸入為 [3,9,3,4,7,2,12] 的陣列，一開始先把左指針擺在最左邊，右指針在最右邊，因為在這情況下可以獲得最大的寬度。而此時的高度必須取最左(3)跟最右(6)之間較小的那個，才能形成完整的矩形，所以此時寬度為 7-0=7 ，高度為 Min(3,6) = 3 ，面積為 7*3 =21。

*(以下的圖都請忽略純粹為了美觀而畫的柱子本體的寬度)*

![](https://cdn-images-1.medium.com/max/2000/1*i84226ZOMg03by12HQk6Jg.png)

所以我們取得了在寬度最大，也就是距離為 7 時的面積大小，接下來要進一步看在寬度縮小之後能否取得更大的面積（若接下來有遇到高度更高的柱子，也可能取得更大面積）。

每次縮小寬度的最小單元是 1，也就是距離會 -1，這時候就需要思考我們應該要移動哪邊的指針，才有可能取得更大面積。若移動右邊指針的情況下：

![](https://cdn-images-1.medium.com/max/2000/1*gK5QHjqCEI1hNoIsbO8rmQ.png)

這時候雖然到達了一個更高高度的柱子(12)，但由於我們的水位高度必須取決於兩根柱子裏面比較短的那根，所以就算獲得更高的柱子也沒用，在這個狀況下高度仍是 3，但由於寬度變小，反而面積更小了。

那如果移動左邊的指針呢？

![](https://cdn-images-1.medium.com/max/2000/1*GcbP2CmUfC9hxFDQhAZcxg.png)

這時可以發現到，由於左邊移動到了 index 1 高度為 9 的柱子，比原本的 index 3 高度為 3 的值還要大，所以在這情況下，取得了一個更大的面積 6*6 = 36。

到這裡我們可以整理出一個規則，當寬度會逐漸變小的情況下，如果想要取得尋更大的面積，高度必須增加才有可能。**而高度若要增加，決定高度的較短柱子需替換成更高的柱子。也就是說，需從較短的柱子那邊開始（往右或往左）移動尋找比其還高的柱子**。

若從較高柱子那邊開始移動，就會出現剛剛[圖2]的情況，無法獲得更大的面積。了解到這個關鍵之後，就可以開始實作了～

## 實作程式碼

尋找下一根要移動到的柱子的部分，用了 while 迴圈的方式直接找往右或往左更高的柱子，可以省下去計算不可能比現有面積還大的時間。

```java
class Solution {
    public int maxArea(int[] height) {
        int left = 0;
        int right = height.length -1;
        int max_area = -1;
        
        while (left < right) {
            
            int w = right - left;
            int h = Math.min(height[left], height[right]);
        
            int new_area = w * h;
            
            if (w * h > max_area) {
                max_area = new_area;
            }
            

	    // 找尋下一根要移動到的柱子
	    // 若 左邊柱子為較小值才會進到這個 while，要往右找到比左邊柱子還高的下一根柱子
            while (left < right && height[left] <= h) {
                left += 1;
            }
	    // 若 右邊柱子為較小值才會進到這個 while，要往左找到比右邊柱子還高的下一根柱子
            while (left < right && height[right] <= h) {
                right -= 1;
            }

        }
        
        return max_area;
    }
}
```
## 複雜度分析

時間複雜度：O(n)/空間複雜度：O(1)
