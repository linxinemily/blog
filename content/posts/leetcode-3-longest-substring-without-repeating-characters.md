---
title: "[leetcode 筆記] 3. Longest Substring Without Repeating Characters"
date: 2022-05-07
tags:
- Algorithm
---

[題目連結](https://leetcode.com/problems/longest-substring-without-repeating-characters/)

## 題目簡述

給定一組字串，回傳這組字串中，最長的不重複子字串長度。例如： pwwkew 的最長不重複子字串為 wke ，所以回傳長度為 3。

## 解題思路

這題我最後被 [Accepted](https://leetcode.com/submissions/detail/694812433/) 的版本（在之前提交了很多次錯誤版本…就不細說了）解法的主要思路為：

遍歷字串同時累加當前字串，**若遇到當前字串當中已經存在(重複)的字元，就把當前字串包含重複字前面的所有字元都剔除掉**，然後就可以繼續往下加。

例如， ohvhjdml 當中，遍歷到 i = 3 時，遇到了第一個重複字元為 “h” （此時的當前字串為 ohv ，最大長度為 3），這時候就把 ohv 當中 ”h” 之前且包含 “h” 的所有字元刪掉，變成剩下 v 之後，再往下加，所以在這個迭代中，當前字串會變成 vh ，最大長度為 2 。

具體實作如下：

```java
class Solution {
    public int lengthOfLongestSubstring(String s) {
        int max_length = 0;
        String longest_str = "";
        
        for (int i = 0; i < s.length(); i++) {
            char c_char = s.charAt(i);
            // 如果沒遇到重複的話，這邊indexOf(c_char)會回傳 -1，而 -1+1 = 0，所以此時字串不會被刪減
            longest_str = longest_str.substring(longest_str.indexOf(c_char) + 1);
            longest_str += c_char;
            max_length = Math.max(max_length, longest_str.length());
        }
        
        return max_length;
    }
}
```

![](https://cdn-images-1.medium.com/max/4756/1*fvlNHY9JgcM0nsm03RFPvg.png)

然而看了一下這結果似乎並非最佳解（但我已然無招QQ），於是參考了下其他人的解法，果然這題是在考一個特定的演算法 — Sliding Window Algorithm，就直接來理解一下使用該演算法優化後的版本吧。

## 優化版解法：Sliding Window + Hash Table

這題其實是 Sliding Window Algorithm 搭配 Hash Table 的應用題型，而 Sliding Window 實作上通常會需要用到雙指針的技巧。

首先，會需要一個 Hash Table （這邊實作上使用 Set 這個資料結構）來紀錄是否有重複的字串，然後一個變數紀錄當前長度最大值，以及 i, j 兩個指針紀錄當前子字串的頭(j)跟尾(i)。

假設同樣以 ohvhjdml 這組字串為例，一開始 i, j 都在 index = 0 的位置， 這時 Set 當中沒有 “o” 這個字元，將 “o” 加入 Set，長度+1。

    //第1次迴圈操作之後的結果
    i = 1 
    j = 0
    max_length = 1
    set = ["o"]

第二次迴圈，i = 1，j =0，這時 Set 當中也沒有 “h” 這個字元，將 “h” 加入 Set。

    //第2次迴圈操作之後的結果
    i = 2 
    j = 0
    max_length = 2
    set = ["o", "h"]

再下一次的 “v” 也是同樣操作，都沒有遇到重複字元的情況下，就是將字元加入 Set，然後前進 i，長度+1。

    //第3次迴圈操作之後的結果
    i = 3
    j = 0
    max_length = 3
    set = ["o", "h", "v"]

接著到了 i = 3，這時 Set 當中已存在 “h”，所以這時要把包含 “h” 之前的字元都從 Set 裡移除（移除 ”o”、”h”），然後把 j 移動到 “v” 的位置，這時 i 跟 j 都在 index = 3 的位置，然後 Set 裏面只剩下 “v”，確保重複字元都移除後，再把後來遇到的那個 “h” 加進 Set，繼續前進 i。

    //第4次迴圈操作之後的結果
    i = 4
    j = 3
    max_length = 3 // max_length 不變
    set = ["v", "h"]

具體實作如下：

```java
class Solution {
    public int lengthOfLongestSubstring(String s) {
        HashSet<Character> c_set = new HashSet<>();
        int max_length = 0, j = 0;
        
        for (int i = 0; i < s.length(); i++) {
            char c_char = s.charAt(i);
            
            while (c_set.contains(c_char)) {
                c_set.remove(s.charAt(j));
                ++j;
            }
            
            c_set.add(c_char);
            max_length = Math.max(max_length, i-j+1);
        }
        
        return max_length;
    }
}
```

到這邊可以發現其實解法邏輯和原本相同，都是在遇到重複字元後，移除子字串當中本來存在重複字元之前的所有字元，只不過優化版使用了特定技巧(Sliding Window/雙指針)和資料結構(Set)省去了複製一份新的子字串以及比較子字串當中重複字元位置這段的成本：

    longest_str.substring(longest_str.indexOf(c_char) + 1);

## 複雜度分析

### 優化版的分析

時間複雜度：最差的情況下， i 和 j 兩個指針會分別遍歷每個字元 O(2n) ，忽略常數的情況下為 O(n)。

空間複雜度：最差的情況下，每個字元都不重複，Set 的大小會等於 n，複雜度為 O(n)。

### 原本作法的分析

時間複雜度和優化版相同，但是多了複製字串還有判斷字元是否重複的步驟，空間複雜度變數都是 scalar ，故為 O(1)（可是Leetcode分析看不太出來差距）。

## 參考資料

優化版的講解影片：[https://www.youtube.com/watch?v=4i6-9IzQHwo&ab_channel=RyanSchachte](https://www.youtube.com/watch?v=4i6-9IzQHwo&ab_channel=RyanSchachte)
