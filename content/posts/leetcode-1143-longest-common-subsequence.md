---
title: "[leetcode 筆記] 1143. Longest Common Subsequence"
date: 2021-07-18
tags:
- Algorithm
---

[題目連結](https://leetcode.com/problems/longest-common-subsequence/)

## 題意簡述

給定兩組字串序列，回傳「Longest Common Subsequence(LCS)」的長度。LCS 的定義為出現於每一個序列、而且是最長的子序列。例如："ace" 和 "abcde" 的 LCS 為 "ace"。

## 解題思路

這題也是 Dynamic Programming 的題目，所以重點在於**找出子問題以及記下子問題的答案。**而要找出子問題通常會透過 Top-down 的方法，也就是先找出 Recursive Solution。

### **Step1 . 找出子問題**

以上面的 "ace" 和 "abcde" 當作例子，首先先觀察這兩個字串序列，並從最後一個字母來觀察，可以發現兩者最後一個字母都是 "e"，這就代表找到了 "ace" 和 "abcde" 的 LCS 其中 1 個值，所以 LCS = 1+ ??? 。

而這個???是什麼呢？可以知道由於目前已經確保 "e" 一定在 LCS 裡面，那就可以先略過這個 e ，那接下來不就是要繼續從 "ac" 跟 "abcd" 當中找出 LCS 嗎，所以這時的 ??? 就是 LCS("ac", "abcd")。

所以在目前在**最後一個字母都相同**的情況下：

    LCS("ace", "abcde") = 1 + LCS("ac", "abcd")

目前的問題變成要解 LCS("ac", "abcd") ，而這兩串序列的最後一個字母不一樣，所以答案可能會是 LCS("a", "abcd") （把 s1 的 c 略過）或是 LCS("ac", "abc") （把 s2 的 d 略過），看哪個結果較大，就取該結果：

在**最後一個字母不同**的情況下：

    LCS("ac", "abcd") = Max(LCS("a", "abcd"), LCS("ac", "abc"))

而繼續往下拆解子問題就會像這樣：

![](https://cdn-images-1.medium.com/max/8098/1*pJ-xqXleIaDLGvdy5zxfrg.png)

其中標黃色的子問題像是 LCS("", "abcd") 、 LCS("", "")，由於空字串不可能會跟其他字串有共同的字母，所以不管遇到什麼字串，空字串與其他字串（包含自己）的 LCS 都會是 0，這也就是遞迴的 Base case，直接回傳 0 結束該回合。

### **Step2 . 記下子問題的答案**

可以發現上面的演算法有許多地方的解是重複的，其實不必每次遇到都重算一次，這也就是 DP 的另一半重點，以 Bottom-up 方式將每個子問題答案都先記下來，以便在之後遇到同樣子問題時可以直接取出返回結果。

以 DP table 為輔助，繼續上面的例子，首先先把 "abcde" 當成縱列，"ace" 當成橫列：

![](https://cdn-images-1.medium.com/max/2100/1*OpAGukX7nA8lvJdVo5Om1g.png)

此時每一格代表的就是每個子問題的解，EX: 第 1 列第 1 行的格子 = LCS("", "")也就是 0，第 4 列第 6 行的格子= LCS("ace", "abcde") 也就是本題要求的答案。

    第 m 列第 n 行的格子 = LCS("ace" 的前 m 個字串, "abcde" 的前 n 個字串)

**以下用 (m,n)代表 「第 m列第 n行的格子」。*

那首先就可以繼續來填這個表格，剛剛已知 (1,1) 為 0，那 (1,2) 就是代表 LCS("a", "") ， (1,3) 代表 LCS("ac", "") ，而一開始已經定義了 Base case，**空字串與任何字串（包含空字串本身）的 LCS 都會是 0**，所以這一排全部都可以填 0 ，同理，縱軸的部分也是一樣，遇到 "" 時答案都是 0。所以表會變這樣：

![](https://cdn-images-1.medium.com/max/2104/1*bQ1MptwwRiXKQgIND2_AfQ.png)

接著看 (2,2)，求 LCS("a", "a") ，尾巴兩個字母相同時，LCS("a", "a") = 1 + LCS("", "") ，而 LCS("", "") 的答案是就在左斜上格 = 0，所以這格就會是 1+0 = 1。

接著看 (2,3)，求 LCS("ac", "a") ，尾巴兩個字母不同，此時LCS("ac", "a") = Max(LCS("a", "a"), LCS("ac", "")) ，可以發現LCS("a", "a") 就是剛剛求過的左邊的格子，而LCS("ac", "") 就是上方的格子。所以只要在左邊和上方的格子中選一個較大的值填入即可，這格的值為 Max(1, 0)。

所以其實我們已經找到了填表的規律：

    若該格子對應的兩個字母相同，答案 = 1 + 左斜上格
    若該格子對應的兩個字母不相同，答案 = Max(左邊那格, 上面那格)

所以其實一切都跟 Step .1 當中遞迴求子問題的邏輯一模一樣，只是換成迭代的形式由下而上求出來而已。

## 程式碼

看到這邊其實程式碼已經不重要了（？），但還是貼一下，實作的二維陣列會從剛剛的表格的 (2,2) 開始，因為反正前面都是 0 ，沒必要多記：

```js
    var longestCommonSubsequence = function(text1, text2) {
        const res = []
        for (let m = 0; m < text1.length; m++) {
            res.push([]) // 先把二維陣列(表格)做出來
        }
        for (let i = 0; i < text1.length; i++) {
            for (let j = 0; j < text2.length; j++) {
                if (text1[i] === text2[j]) {
                   res[i][j] = (i && j ? res[i-1][j-1] : 0) + 1
                } else {
                   res[i][j] = Math.max(
                    i > 0 ? res[i-1][j] : 0,
                    j > 0 ? res[i][j-1] : 0
                   )
                }
            }
        }
        return res[text1.length-1][text2.length-1]
    };
```
## 時間/空間複雜度

* 時間複雜度為 O(mn) ：input 兩個字串的長度分別為 m,n，總執行 m*n 次（先把表格做出來的部分會變多加一個 m，但常數就忽略不計）

* 空間複雜度為 O(mn)：根據 m * n 決定 Array 的大小

## 參考資料

內容多是參考第一個影片，講解得很清楚，覺得非常有幫助!

1. [Longest Common Subsequence (2 Strings) — Dynamic Programming & Competing Subproblems](https://www.youtube.com/watch?v=ASoaQq66foQ)

1. [Longest Common Subsequence (Dynamic Programming)](https://www.youtube.com/watch?v=Qf5R-uYQRPk)
