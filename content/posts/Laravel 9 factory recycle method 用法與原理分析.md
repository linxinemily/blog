---
title: Laravel 9 factory recycle method 用法與原理分析
date: 2022-10-05
description: 介紹 Laravel factory recycle 方法的使用情境及欲解決之問題，並分析原始碼是如何實現這個功能的。
tags:
- Laravel
---

## `recycle` 方法的使用情境及欲解決之問題

在 laravel 9 factory 的[文件當中最後一段](https://laravel.com/docs/9.x/eloquent-factories#recycling-an-existing-model-for-relationships)是在說明 `recycle` 的使用情境與方法：

> If you have models that share a common relationship with another model, you may use the **`recycle`** method to ensure a single instance of the related model is recycled for all of the relationships.<br />
> For example, imagine you have **`Airline`**, **`Flight`**, and **`Ticket`** models, where the ticket belongs to an airline and a flight, and the flight also belongs to an airline. When creating tickets, you will probably want the same airline for both the ticket and the flight, so you may pass an airline instance to the **`recycle`** method:


大致上是說在遇到以下類型的關聯時

![Untitled](images/factory-recycle-method.png)

一個 ticket 屬於一個 flight 與 一個 airline（ticket 身上有 `flight_id` 與 `airline_id`)，而 ticket 所屬於的 flight （flight 身上有 `airline_id`）也會屬於和 ticket 相同的 airline （ticket 的`airline_id` 和 flight 的 `airline_id` 相同）

可以使用 `recycle` 方法來傳入會共用的 airline model 實體：

```php
Ticket::factory()
    ->recycle(Airline::factory()->create())
    ->create();
```

ok，然後這段解說就結束了。我看了好多遍還是滿頭疑惑。直到查到了這個[原始的 PR](https://github.com/laravel/framework/pull/44107)，看了裡面的範例程式碼才理解了 `recycle` 的用意。

PR 的作者展示了在沒有 `recycle` 方法時，如果要建一個滿足需求（ticket 屬於 flight 也屬於 airline，且 flight 的 airline 和 ticket 的 airline 相同）的 Ticket 會需要這樣寫：

```php
$airline = Airline::factory()->create();
Ticket::factory()
	->for($airline)
	->for(Flight::factory()->for($airline))
	->create();
```

但有了 `recycle` 之後，可以改成這樣寫：

```php
$airline = Airline::factory()->create();
Ticket::factory()
	->recycle($airline)
	->create();
```

等於 `recycle` 幫我們把 ticket 所屬於的 airline ，以及 ticket 所屬的 flight 所屬的 airline 都指定成當成參數傳入 `recycle` 方法的那一個 airline 實例。
也就是說在建立 ticket 的過程中，當需要建立其關聯（關聯的關聯、關聯的關聯的關聯⋯⋯）model 時，只要遇到 "BelongsTo airline" 的關聯，都會直接使用該 airline 實例。

測試一下是不是符合我們的想像，首先先開出這三個 Model 與 migration 後，再分別定義三者的 Factory：

##### Airline

```php
class Airline extends Model
{
    use HasFactory;

    public function tickets()
    {
        return $this->hasMany(Ticket::class);
    }

    public function flights()
    {
        return $this->hasMany(Flight::class);
    }
}
```

```php
// airline migration

// default
public function up()
{
    Schema::create('airlines', function (Blueprint $table) {
        $table->id();
        $table->timestamps();
    });
}
```

```php
// AirlineFactory

...
public function definition()
{
    return [
        //
    ];
}
```

##### Flights

```php
class Flight extends Model
{
    use HasFactory;

    public function tickets()
    {
        return $this->hasMany(Ticket::class);
    }

    public function airline()
    {
        return $this->belongsTo(Airline::class);
    }
}
```

```php
// flight migration
...
public function up()
{
    Schema::create('flights', function (Blueprint $table) {
        $table->id();
        $table->timestamps();
        $table->foreignId('airline_id')->constrained();
    });
}
```

```php
// FlightFactory
...
public function definition()
{
    return [
        'airline_id' => Airline::factory()
    ];
}
```

##### Tickets

```php
class Ticket extends Model
{
    use HasFactory;

    public function flight()
    {
        return $this->belongsTo(Flight::class);
    }

    public function airline()
    {
        return $this->belongsTo(Airline::class);
    }
}
```

```php
// tickets migration
...
public function up()
{
    Schema::create('tickets', function (Blueprint $table) {
        $table->id();
        $table->timestamps();
        $table->foreignId('flight_id')->constrained();
        $table->foreignId('airline_id')->constrained();
    });
}
```

```php
// TicketFactory
...
public function definition()
{
    return [
        'flight_id' => Flight::factory(),
        'airline_id' => Airline::factory()
    ];
}
```

tinker 執行結果：

```php
>>> $ticket = App\Models\Ticket::factory()->recycle(App\Models\Airline::factory()->create())->create();
=> App\Models\Ticket {#4619
     flight_id: 1,
     airline_id: 1,
     updated_at: "2022-10-05 11:02:54",
     created_at: "2022-10-05 11:02:54",
     id: 1,
   }

>>> $ticket->airline
=> App\Models\Airline {#3647
     id: 1,
     created_at: "2022-10-05 11:02:54",
     updated_at: "2022-10-05 11:02:54",
   }

>>> $ticket->flight->airline
=> App\Models\Airline {#3628
     id: 1,
     created_at: "2022-10-05 11:02:54",
     updated_at: "2022-10-05 11:02:54",
   }
```

確實 ticket 的 airline 和 ticket 的 flight 的 airline 都是 id 為 1 的 airline。

## 原始碼分析

接著來看一下這個神奇的功能是如何被實現的。

首先 `recyle` 方法會將要共用的實例暫存在 Factory 當中的 `recyle` 屬性當中：

```php
//vendor/laravel/framework/src/Illuminate/Database/Eloquent/Factories/Factory.php L627
public function recycle($model)
{
    // Group provided models by the type and merge them into existing recycle collection
    return $this->newInstance([
        'recycle' => $this->recycle
            ->flatten()
            ->merge(
                Collection::wrap($model instanceof Model ? func_get_args() : $model)
                    ->flatten()
            )->groupBy(fn ($model) => get_class($model)),
    ]);
}
```

`create` 方法被呼叫後，進到 `make` ， 再進到 `$this->count === null` if 敘述裡面 

```php
//vendor/laravel/framework/src/Illuminate/Database/Eloquent/Factories/Factory.php
public function make($attributes = [], ?Model $parent = null)
{
    if (! empty($attributes)) {
        return $this->state($attributes)->make([], $parent);
    }

    if ($this->count === null) {
        return tap($this->makeInstance($parent), function ($instance) { // here
            $this->callAfterMaking(collect([$instance]));
        });
    }
...
}
```

接著進到 `makeInstance` ，當中 `newModel` 時呼叫了 `getExpandedAttributes` 方法

```php
protected function makeInstance(?Model $parent)
{
    return Model::unguarded(function () use ($parent) {
        return tap($this->newModel($this->getExpandedAttributes($parent)), function ($instance) {
            if (isset($this->connection)) {
                $instance->setConnection($this->connection);
            }
        });
    });
}
```

將 `$this->getRawAttributes($parent)` 返回值（我們剛剛在每個 Model Factory 當中的 `defination` 方法回傳的陣列）當作參數塞進 `expandAttributes` 並回傳結果

```php
protected function getExpandedAttributes(?Model $parent)
{
    return $this->expandAttributes($this->getRawAttributes($parent));
}
```

重點就在這裡，還記得前面所建立的 Model Factory 當中我們將 foreign key 分別指定為對應的 Factory

```php
// TicketFactory
...
public function definition()
{
    return [
        'flight_id' => Flight::factory(),
        'airline_id' => Airline::factory()
    ];
}
```

在執行到

`App\Models\Ticket::factory()->recycle(App\Models\Airline::factory()->create())->create()` 

的第二個 create （ `TicketFactory::create`）並且進到 `expandAttributes` 方法裡，當中的 `$definition` 其實就等同上面這個陣列內容（詳情可見 `getRawAttributes` 方法），所以 `$attribute` 分別會是 `Flight::factory()` 和 `Airline::factory()`

而執行第一個迭代時（ `$this` = `TicketFactory`, `$attribute` = `Flight::factory()` ）由於在暫存的 recycle 屬性中找不到對應的 Flight Model，所以將 `Flight::factory()` 也傳入當前的 `recycle` 屬性後再 create。就是在這個地方形成遞迴，不斷把 recycle 傳給下一個屬於的關聯後再建立關聯 Model。

```php
 protected function expandAttributes(array $definition)
{
    return collect($definition)
        ->map($evaluateRelations = function ($attribute) {
            if ($attribute instanceof self) {
                ///***
                $attribute = $this->getRandomRecycledModel($attribute->modelName())
                    ?? $attribute->recycle($this->recycle)->create()->getKey();
                ///***
            } elseif ($attribute instanceof Model) {
                $attribute = $attribute->getKey();
            }

            return $attribute;
        })
...
}
```

## 總結

之所以會有 `recycle` 這個方法，主要是為了可以使用同一個實例作為關聯且不需巢狀呼叫 `for` 方法。
重點在於 **需事先在 Model Factory 當中將 `{model}_id`指定為對應的 Factory (`{model}::factory()`)** ，要達到這點 `recycle` 才會有作用。

個人是覺得文件沒有寫得非常清楚，花了一番力氣才真正理解它的用法，但也剛好可以理解一下原理啦。
