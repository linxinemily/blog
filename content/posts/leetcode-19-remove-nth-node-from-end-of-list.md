---
title: "[leetcode 筆記] 19. Remove Nth Node From End of List"
date: 2022-05-25
tags:
- Algorithm
---

[題目連結](https://leetcode.com/problems/remove-nth-node-from-end-of-list/)

## 題意簡述

給一個單向 Linked list 的 head，以及一個數字 n ，將倒數第 n 個 linked list 中的節點移除後，回傳該 linked list 的 head。

以題目給的範例 input 為例，Linked list: [1,2,3,4,5]，n: 2，需將倒數第 2 個節點也就是(4)移除後回傳 head。

![數字代表節點裡存的值](https://cdn-images-1.medium.com/max/2000/0*iAckbsXrLGMe4Uxl.jpg)*數字代表節點裡存的值*

## 解題思路

在單向 Linked list 當中，「刪除」其實就是更新即將被「刪除」的節點的前一個節點的 next 值，將他指到即將被「刪除」的節點的 next（有點繞口令）。

由於題目要求移除的元素是倒數第 n 個節點，代表我們必須先到達 list 的最後一個節點，然後才能再從最後面開始往回數。

所以這時候可以利用到遞迴的特性，採取 DFS 走到底的探測方式，先走到 Linked list 的最後一個元素。

當到達最後一個元素後，遞迴會開始收斂回來，往回走的過程再一邊計算當前是位於倒數第幾個元素，若數字剛好等於 n 時，就回傳該節點的下一個節點，否則回傳當前節點。

```go
/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */
func removeNthFromEnd(head *ListNode, n int) *ListNode {
    count_from_last := 0
    return removeHelper(head, n, &count_from_last)
}

func removeHelper(node *ListNode, n int, count_from_last *int) *ListNode {
    
    if node == nil {
        return nil
    }
    node_next := removeHelper(node.Next, n, count_from_last)
  
    //走到最底後會開始往回走，所以可以接著做其他事

    node.Next = node_next // 更新 next(只有被刪除節點會回傳不同於原本的 next)
    
    *count_from_last++
    
    if *count_from_last == n {
        return node_next
    } 
    
    return node
}
```
當走到最後一個節點 (5) 時，由於(5)的 next 為 nil，所以這時遞迴會開始往回收斂

    current node: 5
    next node: null (前一個遞迴回傳值)

    更新 5 的 next 為 null

    current count: 1 // != 2

    回傳當前節點 5 

接著回到 (4)，這時剛好數到倒數第 2 個節點，所以回傳前一個節點

    current node: 4
    next node: 5 (前一個遞迴回傳值)

    更新 4 的 next 為 5

    current count: 2 // 正好等於 2!!

    回傳前一個節點 5

接著回到 (3)，由於剛剛執行 (4) 時回傳的值為 (5) ，所以這時更新 (3) 的 next 為 (5)，成功刪除 4

    current node: 3
    next node: 5(前一個遞迴回傳值)

    更新 3 的 next 為 5  <--- 成功刪除 4

    current count: 3 // != 2

    回傳當前節點 3

總之，大致上就是先確定終止條件以及遞迴呼叫的流程，再思考到達終止條件後接下來往回收斂的部分要做什麼，像是“回傳值要如何被使用”以及“更新 next 的時機點”的部分都是邊寫邊想出來的。

## 複雜度分析

時間複雜度：O(n)/空間複雜度O(1)
