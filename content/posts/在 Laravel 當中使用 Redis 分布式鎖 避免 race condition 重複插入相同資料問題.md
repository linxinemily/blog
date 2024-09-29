---
title: 在 Laravel 當中使用 Redis 分布式鎖 避免 race condition 重複插入相同資料問題
date: 2022-11-06
tags:
- Laravel
- Redis
---

前幾天遇到一個狀況，原本程式裡有一段邏輯是 **「如果該筆資料不存在，就寫入新資料，但如果已存在，就直接回傳該筆已存在的資料」**，程式碼大概長這樣：

```php
$snapshot = OrderSnapshot::where('order_id', $order->id)->first();
if (!empty($snapshot)) {
  return $snapshot;
}

// do something...

return OrderSnapshot::create(['payload' => $order_array]);
 

```

上面程式碼其實也可以使用 Laravel 的 `firstOrCreate` 方法做簡化:
```php
return OrderSnapshot::firstOrCreate(
  ['order_id' => $order->id],
  ['payload' => $order_array]
)
```

這段邏輯看起來很簡單，在單一 process 的情況下，這段程式碼確實沒有任何問題，但如果今天同時有兩個 process 要執行這段程式的話，就可能會有非預期的結果出現。
由於不論是上面的程式碼，或是簡化後的程式碼，如果沒有撈出資料的話（也就是資料不存在），都是要繼續往下執行新增資料的動作，而「撈出資料」和「新增資料」都是單獨的 Query，所以如果幾乎同時執行這段程式時，可能就會產生兩筆相同的資料：

![Untitled](images/laravel-redis-lock.png)

而要解決這個問題，大概有以下幾種辦法：

#### 1. 避免多個 process 執行
如果是因為放在 job 裡面，同時有多個 queue worker 在消化這些 job，就保持 queue worker 的數量為 1。
但我們這次遇到的情況並不適用，因為還可以透過 API Request realtime 執行這段程式。而且如果遇到真的很耗時的工作，也不可能為此只開一個 worker。

#### 2. 使用 `INSERT INTO .. ON DUPLICATE KEY` 語法
也就是說將兩個 Query 合併成一個 Query，讓資料庫去判斷資料存不存在，不存在就新增，存在則改成更新。
但這又不太符合我們的情境，因為如果已存在，並不需要對該筆資料做更新，直接取出即可。

#### 3. 利用外部服務實現分布式鎖機制
由於 PHP 單執行緒的特性，本身沒有這種機制，所以只能藉由外部的服務來實現分布式鎖的功能， 像是 Redis 就是一個常被拿來做分布式鎖的工具，而 Laravel 本身有提供 Redis Lock 相關的 API，所以最後採用了這個解法。

我們要達到的目的其實很簡單，也就是要把「撈出資料」和「新增資料」這兩個操作視為一個原子性的操作，如果第一個程序還沒執行完，第二個程序必須等待第一個做完後，才能接著做。利用分布式鎖的話就能夠實現這個行為：
當某一個程序還在執行這些動作的時候（還拿著這個鎖），其他程序都必須等待它執行完（釋放鎖）之後才能接著執行（獲得鎖）。如此一來就不會發生產生相同資料的問題。


在使用 Laravel Redis Lock 之前，先看一下在 Redis 當中是怎麼實現 Lock 機制的。


## Redis Lock 機制


首先，在 redis-cli 下一個指令取得鎖
```
SET resource_name my_random_value NX PX 30000
```
* `resource_name`：鎖的 Key

* `my_random_value`：鎖的 Value，實作上通常會設置一串 Random String，當作取得該鎖的 Client 識別符，在釋放鎖時需檢查 Key 所對應的 Value 是不是和取得鎖時所設置的 Value 相同，避免鎖被其他 Client 釋放

* `NX`：只有當 `resource_name` 不存在才創建這個 Key

* `PX 30000`： Key 的有效期間是 30000 毫秒，超過會自動失效


就這樣！而「釋放鎖」的實作，通常會寫成一個 LUA 腳本，給 Redis 來執行：
```lua
if redis.call("GET",KEYS[1]) == ARGV[1]
then
    return redis.call("DEL",KEYS[1])
else
    return 0
end
```

接著我們可以回到 Laravel 當中去使用 Redis Lock 了！

## Laravel Redis Lock

### 建立並取得鎖

首先要建立一個鎖，要先 new 一個 `Illuminate\Cache\RedisLock`

> `Illuminate\Cache\RedisLock` 實作 `Illuminate\Contracts\Cache\Lock` interface，實現取得鎖、釋放鎖等功能，並繼承 `Illuminate\Cache\Lock` abstract class 共享分布式鎖的相同功能。

```php
use Illuminate\Cache\RedisLock;
use Illuminate\Support\Facades\Redis;

$lock = new RedisLock(Redis::connection(), "creating:snapshot:$order->id", 15);

```
來看一下 `Illuminate\Cache\RedisLock` 的建構子寫了什麼
```php
// vendor/laravel/framework/src/Illuminate/Cache/RedisLock.php

/**
 * Create a new lock instance.
 *
 * @param  \Illuminate\Redis\Connections\Connection  $redis
 * @param  string  $name
 * @param  int  $seconds
 * @param  string|null  $owner
 * @return void
 */
public function __construct($redis, $name, $seconds, $owner = null)
{
 parent::__construct($name, $seconds, $owner);

 $this->redis = $redis;
}
```
* `$name` ：鎖的 Key。在我們的例子當中，Key 值應該會是 OrderSnapshot 的 order_id，因為我們就是為了避免相同 order_id 的 OrderSnapshot 被建立，所以要處理特定 order_id 時，應該要取得一個鎖，避免其他執行序也對相同 order_id 做處理。

* `$seconds`：Key 的有效期間
* `$owner`： 鎖的 Value。如果不帶該參數，Laravel 會幫我們產生一串亂數當成 Value。在 `Illuminate\Cache\Lock` 的建構子當中：
```php
// vendor/laravel/framework/src/Illuminate/Cache/Lock.php
public function __construct($name, $seconds, $owner = null)
{
 if (is_null($owner)) {
   $owner = Str::random();
 }

 $this->name = $name;
 $this->owner = $owner;
 $this->seconds = $seconds;
}
```

然後呼叫 `acquire` 方法時，才會真的向 Redis 下指令建立鎖 

```php
// vendor/laravel/framework/src/Illuminate/Cache/RedisLock.php

/**
* Attempt to acquire the lock.
*
* @return bool
*/
public function acquire()
{
 if ($this->seconds > 0) {
   return $this->redis->set($this->name, $this->owner, 'EX', $this->seconds, 'NX') == true;
 } else {
   return $this->redis->setnx($this->name, $this->owner) === 1;
 }
}

```

### 釋放鎖
同樣定義在 `RedisLock` 的實作裡，使用了剛剛提到的 Lua 腳本，傳給 Redis 去執行，腳本內容就是同樣的內容，這邊就不貼了
```php
// vendor/laravel/framework/src/Illuminate/Cache/RedisLock.php

/**
 * Release the lock.
 *
 * @return bool
 */
public function release()
{
  return (bool) $this->redis->eval(LuaScripts::releaseLock(), 1, $this->name, $this->owner);
}
```


取得鎖和釋放鎖的流程應該會類似這樣(偽代碼)：
```php
while {

 if ($lock->acquire()) {
  // 處理業務邏輯

  $lock->release();

  break;
 }


 //maybe wait for seconds?
 sleep();

}

```

而 `RedisLock` 當中有一個 `block` 方法可以使用，已經幫我們處理了流程控制的部分，還能額外設置等待釋放鎖的超時時間（超過會拋出異常）：

```php

public function block($seconds, $callback = null)
{
 $starting = $this->currentTime();

 while (! $this->acquire()) {
   //  - 7.x 之後的 laravel 會有 sleepMilliseconds 參數，預設為 250 毫秒
   //  - 7.x 以前的直接寫死 250 毫秒
   usleep($this->sleepMilliseconds * 1000); 
   

   if ($this->currentTime() - $seconds >= $starting) {
     throw new LockTimeoutException;
   }
 }

 // 取得鎖之後執行 callback 並釋放鎖
 if (is_callable($callback)) {
   try {
     return $callback();
   } finally {
     $this->release();
   }
 }

 return true;
}

```


最後使用 `block` 方法，改寫我們一開始的程式碼：
```php
$lock = new RedisLock(Redis::connection(), "creating:snapshot:$order->id", 15);

return $lock->block(5, function () use ($order) {

  $snapshot = OrderSnapshot::where('order_id', $order->id)->first();
  
  if (!empty($snapshot)) {
    return $snapshot;
  }

  // do something...

  return OrderSnapshot::create(['payload' => $order_array]);
});


```