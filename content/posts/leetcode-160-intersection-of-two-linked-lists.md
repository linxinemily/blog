---
title: "[leetcode 筆記] 160. Intersection of Two Linked Lists"
date: 2023-07-31T23:54:43+08:00
tags:
- Algorithm
---

## 題目概述
給定兩個單向 Linked List 的第一個節點 (head)，兩個 List 可能會在某個節點會合成為一個 List，要求找出會合節點並回傳。若兩個 List 並無會合，則回傳 null。

![160. Intersection of Two Linked Lists](images/leetcode-160-intersection-of-two-linked-lists.png)

其中，會合節點必須是指向同一個實例（指標），也就是即使兩個節點的值相同，但為不同實例，不能算是會合節點。

[題目連結](https://leetcode.com/problems/intersection-of-two-linked-lists/description/)
## 解法一：HashMap

也是第一時間想到的方法，先遍歷其中一個 List，使用一個 Hashmap 紀錄 key 為實例指標，值為節點的值（其實值為何並不重要），接著再遍歷第二個  List，判斷當前節點是否存在於 map 當中，有的話則立即返回，都遍歷完的話代表兩個 List 並無相同元素，也就是不存在會合節點。

```go
func getIntersectionNode(headA, headB *ListNode) *ListNode {
  m1 := make(map[*ListNode]int)
  curA := headA
  for curA != nil {
    m1[curA] = curA.Val
    curA = curA.Next
  }

  curB := headB
  for curB != nil {
    if _, ok := m1[curB]; ok {
      return curB
    }
    curB = curB.Next
  }

  return nil
}
```

但這個方法其實會需要 O(n) 的空間複雜度，雖然可以被 Accepted，但並非最佳解。

> 無法滿足 follow up 要求：空間複雜度 O(1)，時間複雜度 O(m+n)
> 

## 解法二：雙指針

送出後滑了一下 Solutions 當中其他強者的解法，才看到這個聰明的方法。

配置兩個指針，都各自從 List A 和 List B 的 head 開始，指針1 將 List A 走到底後接著走 List B，同時，指針2 將 List B 走到底後接著走 List A。短的 List 會被先走完就會換到長的 List 去，而長的 List 走完也會換到短 List 去走，最終兩個指針就會走到交會的節點上。

做了個簡單的動畫，直接看比較清楚：

![Intersection of Two Linked Lists 雙指針解法動畫](images/leetcode-160-intersection-of-two-linked-lists.gif)


不管是先走完 A 再走 B，還是先走完 B 再走 A，當走到互換 List 那邊去之後，會在面臨交會點之前同步：

> `a1 → a2` → `c1 → c2 → c3` → `null`(a走完換b) → `b1 → b2 → b3` → `c1`
>
> `b1 → b2 → b3` → `c1 → c2 → c3` → `null`(b走完換a)  → `a1 → a2` → `c1`

 `b3` 的 next 為 `c1` ， `a2` 的 next 也為 `c1` ，得出交會點為 `c1` 。


那如果當沒有交會點的時候會發生什麼事？用同樣的思考方式來看：

> `a1 → a2 → a3 → a4` → `null`(a走完換b) → `b1 → b2 → b3` → `null`
>
> `b1 → b2 → b3` → `null`(b走完換a) → `a1 → a2 → a3 → a4` → `null`

由於 `b3` 和 `a4` 的 next 都是 null，得出交會點為 null。

> 如果List A 和 List B 的長度相同，則此解決方案將在不超過1次遍歷的情況下終止；如果兩個 List 的長度不同，則此解決方案將在不超過2次遍歷的情況下終止 -- 在第二次遍歷中，交換 A 和 B 會使 A 和 B 在第二次遍歷結束前同步。在第二次遍歷中，它們都具有相同的剩餘步驟，以確保它們同時（從技術上來說，是在同一次迭代中）到達第一個交會節點，或者同時到達 null。
by [Java solution without knowing the difference in len!](https://leetcode.com/problems/intersection-of-two-linked-lists/solutions/49785/java-solution-without-knowing-the-difference-in-len/) 

最後的 code：

```python
class Solution:
    def getIntersectionNode(self, headA: ListNode, headB: ListNode) -> Optional[ListNode]:
      p1 = headA
      p2 = headB
      while(p1 != p2):
        p1 = headB if p1 is None else p1.next
        p2 = headA if p2 is None else p2.next
      return p1 
```

## Reference
* [Java solution without knowing the difference in len!](https://leetcode.com/problems/intersection-of-two-linked-lists/solutions/49785/java-solution-without-knowing-the-difference-in-len/)：
這篇 Solution 底下有一個 BryanBoCao 大大的（應該是第一個）留言，當中有詳細的圖解說明，非常有助於理解。