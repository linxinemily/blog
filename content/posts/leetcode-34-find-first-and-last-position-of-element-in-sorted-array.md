---
title: "[leetcode 筆記] 34. Find First and Last Position of Element in Sorted Array"
date: 2024-12-15T00:00:00+08:00
tags:
- Algorithm
---

[題目連結](https://leetcode.com/problems/find-first-and-last-position-of-element-in-sorted-array/description/)

## 題意簡述

給定一個排序過的陣列 nums，以及一個目標值 target，要求找出 target 在 nums 中的起始位置與結束位置，若 target 不存在於 nums 中，則回傳 [-1, -1]。

Example 1:
```
Input: nums = [5,7,7,8,8,10], target = 8
Output: [3,4]
```
Example 2:
```
Input: nums = [5,7,7,8,8,10], target = 6
Output: [-1,-1]
```
## 解題思路

這題是屬於二分搜尋法的變形題，原本二分搜尋法是找到 target 後直接回傳：

```python
def binary_search(nums, target):
    left, right = 0, len(nums) - 1
    while left <= right:
        mid = left + (right-left) // 2
        if nums[mid] == target:
            return mid # 找到 target 直接回傳
        elif nums[mid] < target:
            left = mid + 1
        elif nums[mid] > target:
            right = mid - 1
        return -1
```


而這題要找到 target 的起始位置與結束位置，所以我們需要找到 target 的左邊界與右邊界。

最直覺的方法是先找到 target 的位置，再使用線性搜尋，分別往左右擴展直到碰到邊界。但這樣在最差的情況下（target 重複很多次），時間複雜度會變成 O(n)。

因此如果要維持原本二分搜尋法的時間複雜度 O(log n)，我們可以在找到 target 後，繼續進行二分搜尋，直到找到左邊界**或**右邊界。也就是會實施兩次二分搜尋，一次找左邊界，一次找右邊界。

聽起來有點複雜，但其實只是在原本的二分搜尋法上稍作修改而已。

## 解題步驟

##### 1. 先找到左邊界

想要找到左邊界，我們只需要在找到 target 後，縮小搜索區間的右邊界即可。意即當 nums[mid] == target 時，我們將 right 設為 mid - 1，這樣就可以繼續往左搜尋。

```python
def searchRange(nums, target):
    left, right = 0, len(nums) - 1
    while left <= right:
        mid = left + (right - left) // 2
        if nums[mid] == target:
            right = mid - 1 # 找到 target 後縮小右邊界，繼續往左搜尋
        elif nums[mid] < target:
            left = mid + 1
        elif nums[mid] > target:
            right = mid - 1
    
    if left < len(nums) and nums[left] == target: # left 會在過程中遞增，需確保左邊界不會超出範圍
        return left
    
    return -1
```

> 另外由於迴圈結束時，left 會等於 right + 1，所以這邊回傳 right + 1 也是可以的。

##### 2. 再找到右邊界

找到右邊界的方法與找左邊界相似，只是變成在找到 target 後，縮小搜索區間的左邊界。意即當 nums[mid] == target 時，我們將 left 設為 mid + 1，這樣就可以繼續往右搜尋。

```python
def searchRange(nums, target):
    left, right = 0, len(nums) - 1
    while left <= right:
        mid = left + (right - left) // 2
        if nums[mid] == target:
            left = mid + 1 # 找到 target 後縮小左邊界，繼續往右搜尋
        elif nums[mid] < target:
            left = mid + 1
        elif nums[mid] > target:
            right = mid - 1
    
    if right >= 0 and nums[right] == target: # right 會在過程中遞減，需確保右邊界不會超出範圍
        return right
    
    return -1
```
> 同樣地，迴圈結束時，left 會等於 right + 1，所以這邊回傳 left - 1 也是可以的。

##### 3. 結合左右邊界的結果

最後只要將分別尋找左右邊界的結果結合並回傳即可。

```python
def searchRange(self, nums: List[int], target: int) -> List[int]:
        left, right = 0, len(nums) - 1
        resL, resR = -1, -1

        while left <= right:
            mid = left + (right - left) // 2
            if nums[mid] < target: 
                left = mid + 1
            elif nums[mid] > target:
                right = mid - 1
            else:
                right = mid - 1
        
        if left < len(nums) and nums[left] == target:
            resL = left
        
        left, right = 0, len(nums) - 1
        while left <= right:
            mid = left + (right - left) // 2
            if nums[mid] < target:
                left = mid + 1
            elif nums[mid] > target:
                right = mid - 1
            else:
                left = mid + 1
        
        if right >= 0 and nums[right] == target:
            resR = right

        return [resL, resR]
```

## 複雜度分析

- 時間複雜度：O(log n)，使用兩次二分搜尋法 (O(log n) + O(log n))，所以時間複雜度為 O(log n)。
- 空間複雜度：O(1)，只使用了常數空間。