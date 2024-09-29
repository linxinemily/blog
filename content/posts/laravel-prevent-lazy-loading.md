---
title: Laravel Prevent Lazy Loading 踩坑心得及原理分析
date: 2022-12-29
tags:
- Laravel
---

在 Laravel 當中，我們可以透過「[動態關聯屬性(Dynamic relationship properties)](https://laravel.com/docs/9.x/eloquent-relationships#relationship-methods-vs-dynamic-properties)」直接取得 Model 的關聯資料，而該行為被官方稱為 「Lazy Loading」，當取值時就會自動將關聯資料載入，非常方便。但使用上一不小心就很有可能會造成 N+1 問題。

> N+1 問題通常是指，在得到一個 Models of Collection/Array 後，又在遍歷每個 Model 時，透過 Lazy Loading 取得其關聯資料，會導致 有 N 個 Model 就會執行 N+1 次 Query。過多的 DB Query 次數可能會對效能造成嚴重影響。

Laravel 8 開始就有提供[避免 Lazy Loading 的方法](https://laravel.com/docs/9.x/eloquent-relationships#preventing-lazy-loading)，
只要在 `App\Providers\AppServiceProvider` 當中的 `boot()` 方法裡面加上：

```php
// app/Providers/AppServiceProvider.php
use Illuminate\Database\Eloquent\Model;

public function boot()
{
    Model::preventLazyLoading(! app()->isProduction()); //預設為 true
}
```

如此一來，當 Lazy Loading 被觸發時，就會直接拋出例外 (`Illuminate\Database\LazyLoadingViolationException`)

而當初該新功能發布時，laravel-news 網站[有一篇文章](https://laravel-news.com/disable-eloquent-lazy-loading-during-development)就在介紹這個功能。該篇文章當中舉的例子為，有兩個 Model - User 和 Posts 為一對多關係：
```php
$user = User::first();

$user->posts; // 這邊就會觸發 Lazy Loading, 將所有 user 的 post 都從 DB 撈出
```

所以如果加上 `Model::preventLazyLoading()` ，執行上面範例時應該就會拋出 `Illuminate\Database\LazyLoadingViolationException` 阻止我們透過 Lazy Loading 得到 posts。

於是打開專案來試試，卻發現一切正常，沒有例外拋出...

![Untitled](images/laravel-lazy-loading-1.png)

但如果換一個方法試試：

![Untitled](images/laravel-lazy-loading-2.png)

這時候才會如預期拋出例外。

另外，如果當 DB 裡只有一筆 user 資料，就算像上面這樣取 `all()`，也不會拋出例外。

![Untitled](images/laravel-lazy-loading-3.png)

一開始很疑惑為什麼範例的 code 執行後沒有拋出例外，所以就研究了一下原始碼的部分，順便看一下這個功能是如何實現的。

## 原始碼分析

首先，從 `Illuminate\Database\Eloquent\Model::preventLazyLoading()` 這個方法開始看起：
```php
public static function preventLazyLoading($value = true)
{
    static::$modelsShouldPreventLazyLoading = $value;
}
```
這邊會將 `Illuminate\Database\Eloquent\Model` 當中的一個靜態屬性 `$modelsShouldPreventLazyLoading` 設值成傳入的布林值。

該屬性預設為 `false`：
```php
protected static $modelsShouldPreventLazyLoading = false;
```

但其實 `Illuminate\Database\Eloquent\Model` 另外還有一個非靜態屬性 `$preventsLazyLoading`（待會會提到在哪裡被使用）
```php
public $preventsLazyLoading = false;
```

當執行 DB Query，取得 `App\Models\User` 時(eg: `User::first();`)，會進入以下方法
```php
//src/Illuminate/Database/Eloquent/Builder.php
public function hydrate(array $items)
{
    $instance = $this->newModelInstance();

    return $instance->newCollection(array_map(function ($item) use ($items, $instance) {
        $model = $instance->newFromBuilder($item);

        if (count($items) > 1) {
            $model->preventsLazyLoading = Model::preventsLazyLoading();
        }

        return $model;
    }, $items));
}
```
如果執行 `User::first()`，此處的 `$items` 引數會是一個陣列，內含一個 stdClass 物件 (從 users table 取出的第一個 row 的資料)，像這樣：

```php
 array:1 [
  0 => {#4352
    +"id": 1
    +"name": "Mrs. Gisselle Sauer"
    +"email": "dion.kiehn@example.com"
    +"email_verified_at": "2022-12-22 03:53:59"
    +"password": "$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi"
    +"remember_token": "jq4qLUs2qQ"
    +"created_at": "2022-12-22 03:53:59"
    +"updated_at": "2022-12-22 03:53:59"
  }
]
```
接著在遍歷陣列的過程中，將遍歷到的 stdClass 物件轉成 Eloquent Model 時，會先判斷如果 `$items` 陣列長度大於 1，才會將這個 Model(在此範例也就是 `App\Models\User`)的 `$preventsLazyLoading` 屬性設為`Model::preventsLazyLoading()` 的值，而這個靜態方法其實回傳的就是先前我們在 `App\Providers\AppServiceProvider` 設定的靜態屬性 `$modelsShouldPreventLazyLoading`：
```php
public static function preventsLazyLoading()
{
    return static::$modelsShouldPreventLazyLoading;
}
```

接著，當呼叫 `$user->posts` 時，會進入 `Illuminate\Database\Eloquent\Concerns\HasAttributes` 當中的 `getRelationValue()` 方法:

```php
// src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
public function getRelationValue($key)
    {
        ...

        if ($this->preventsLazyLoading) { // here
            $this->handleLazyLoadingViolation($key);
        }

        return $this->getRelationshipFromMethod($key);
    }
```
當 `$preventsLazyLoading` 為 `true` 時，就會拋出例外
```php
// src/Illuminate/Database/Eloquent/Concerns/HasAttributes.php
protected function handleLazyLoadingViolation($key)
{
    ...

    throw new LazyLoadingViolationException($this, $key);
}
```


也就是說，**從 DB 取出的資料筆數需大於 1 ，對該 Model 取關聯時才會觸發 Prevent Lazy Loading 的效果**。

所以如果撈出的結果都是一筆資料，就也不會觸發 Prevent Lazy Loading 了。
```php
$user = User::first(); // 只有從 DB 撈出一筆資料
$user->posts; // no exception

$users = User::all(); // 假使 DB 裏其實也只有一筆 user 資料
$users[0]->posts; // no exception

```


其實按照常理，大概能夠理解爲什麼會有這種效果，因為只有一個 Model 的時候，取得其底下關聯，應該不能算是 N+1 問題。（不過該網站的例子為什麼這樣舉就不得而知，算是小踩坑了下）