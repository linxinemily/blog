---
title: Laravel IoC Container 如何解析 Controller method 當中的依賴(DI)
date: 2022-07-18
tags:
- Laravel
---

> 由於以下所分析的原始碼是舊版(5.5)的 laravel，但因為本文主要探討 IoC 在特定 method 的依賴解析，核心概念是不變的，但可能不同版本在部分程式碼上面會有些許差異。

熟悉 Laravel 的人應該都曉得 IoC Container 的強大之處，在 laravel 當中，可以簡單地透過一行 `app(MyClass::class)` 取得已經被注入依賴後 Class 的實例：

```php
class Dependence1 {
    function foo() {
        echo "foo";
    }
}

class Dependence2 {
    function foo2() {
        echo "foo2";
    }
}

final class MyClass
{
    private $dep1;
    private $dep2;

    public function __construct(
        Dependence1 $dependence1,
        Dependence2 $dependence2
    )
    {
        $this->dep1 = $dependence1;
        $this->dep2 = $dependence2;       
    }
   
}

app(MyClass::class); // maybe $this->app->make(MyClass::class) in ServiceProvider
```

這是一個 laravel 實現控制反轉(IoC)的簡單例子 ，MyClass 的建構子當中的兩個引數(依賴的Class) 被 container 讀取到後，先分別被實例化並作為參數傳進 MyClass 的建構子，最後返回實例化完成的 MyClass 物件。(每次拜讀到這邊真的是想跪 laravel 作者）

> IoC 的核心理念就是將原本在下層（具體）的程式碼中「注入依賴」這個行為，提升至上層（抽象）去控制決定要給該 Class 注入什麼依賴，也就是說下層的 Class 需要依賴時，都會向上層去要他所需要的依賴實例（特定 Class 或是 implement 特定 Interface 的實例）。
> 

laravel 當中許多基礎核心的 Class 在底層也是這麼運作的，例如 Controller 也是經由 IoC Container 實例化，所以我們才能夠在 Controller 建構子的引數寫一堆依賴然後開心地使用。

```php
class MyController extends Controller
{
 protected $logger;
 public function __construct(Logger $logger)
 {
 $this->logger = $logger;
 }
 public function index()
 {
    $this->logger->error('Something happened');
 }
}
```

然而，laravel 不只會自動解析在建構子引數當中的 Dependencies，在某些特定 Class 當中，laravel 也會自動解析 method 引數中的依賴，最常見的的就是我們的 Controller：

```php
class MyController extends Controller
{

 public function show(Logger $logger, $id)
 {
    $logger->error('Something happened');
 }
}
```

到底其背後是怎麼實現這個功能的呢？以下就開始 trace code 來看看吧。

首先，一個 Request 的旅程就是從 `Illuminate\Contracts\Http\Kernel` 開始， 在 index.php 當中，實例化 Kernel Class 之後， 呼叫了 handle 這個 method：

```php
// public/index.php

$kernel = $app->make(Illuminate\Contracts\Http\Kernel::class);

$response = $kernel->handle(
    $request = Illuminate\Http\Request::capture()
);
```

可以在 bootstrap/app.php 當中找到該抽象對應的綁定，也就是 `App\Http\Kernel` ：

```php
// bootstrap/app.php 

$app->singleton(
    Illuminate\Contracts\Http\Kernel::class,
    App\Http\Kernel::class
);
```

接著， `App\Http\Kernel` 又 extend 了 `Illuminate\Foundation\Http\Kernel` ，handle 方法就在這個 Class 當中定義，重點是 sendRequestThroughRouter 這個 method，準備要將 Request 分配給 Router (上面都還在 application 層，從這邊開始進到 framework 裡面)：

```php
// vendor/laravel/framework/src/Illuminate/Foundation/Http/Kernel.php

public function handle($request)
    {
        try {
            $request->enableHttpMethodParameterOverride();

            $response = $this->sendRequestThroughRouter($request);
      ...
    }
```

```php
// vendor/laravel/framework/src/Illuminate/Foundation/Http/Kernel.php

protected function sendRequestThroughRouter($request)
    {
        ...

        return (new Pipeline($this->app))
                    ->send($request)
                    ->through($this->app->shouldSkipMiddleware() ? [] : $this->middleware)
                    ->then($this->dispatchToRouter()); // 將 Request 分配給 Router
    }
```

```php
// vendor/laravel/framework/src/Illuminate/Foundation/Http/Kernel.php

protected function dispatchToRouter()
    {
        return function ($request) {
            $this->app->instance('request', $request);

            return $this->router->dispatch($request);
        };
    }
```

接下來要進入到 Router 的階段，有些不是很相關的部分就直接跳過不貼程式碼了，進到 Router 後的 method 呼叫順序是 dispatch  → dispatchToRoute → runRoute -> runRouteWithinStack

來看一下 在 runRouteWithinStack 當中倒數第三行 `$route->run()` ，這裡即將要準備解析在 routes/xxx.php 當中所定義的 Route 所指定的對應 Controller 以及 Controller method：

```php
// vendor/laravel/framework/src/Illuminate/Routing/Router.php

protected function runRouteWithinStack(Route $route, Request $request)
    {
        ...
        return (new Pipeline($this->container))
                        ->send($request)
                        ->through($middleware)
                        ->then(function ($request) use ($route) {
                            return $this->prepareResponse(
                                $request, $route->run()
                            );
                        });
    }
```

這邊進入了 Route，主要在準備執行 Route 指定對應的 Controller 以及 Controller method，進到 `$this->runController()` 的部分：

```php
// vendor/laravel/framework/src/Illuminate/Routing/Route.php
public function run()
    {
        $this->container = $this->container ?: new Container;

        try {
            if ($this->isControllerAction()) {
                return $this->runController(); // !!!
            }

            return $this->runCallable();
        } catch (HttpResponseException $e) {
            return $e->getResponse();
        }
    }
```

這裡會呼叫到 ControllerDispatcher 的 dispatch 方法接手處理：

```php
// vendor/laravel/framework/src/Illuminate/Routing/Route.php

protected function runController()
    {
        return $this->controllerDispatcher()->dispatch(
            $this, $this->getController(), $this->getControllerMethod()
        );
    }
```

挖了這麼深，終於要到重頭戲了，也就是呼叫了 `$this->resolveClassMethodDependencies` 這個 method，並將路由上面所帶的參數、已實例化後的 Controller 以及 method 名稱做為參數傳入：

```php
// vendor/laravel/framework/src/Illuminate/Routing/ControllerDispatcher.php

public function dispatch(Route $route, $controller, $method)
    {
        $parameters = $this->resolveClassMethodDependencies(
            $route->parametersWithoutNulls(), $controller, $method
        );

        ...
        return $controller->{$method}(...array_values($parameters));
    }
```

resolveClassMethodDependencies  呼叫了 resolveMethodDependencies，並將剛剛的路由參數以及一個新的 ReflectionMethod 實例做為參數傳入，原來實現 Method Dependency injection 的重要功臣就是 ReflectionMethod 這個東西，藉由 ReflectionMethod，我們可以獲得某個 instance 裡面關於某個 method 的資訊，包括**可以知道他有哪些引數**，而這也就是如何得以偵測到這些 method 需要的依賴，並且預先解析好物件當真正要呼叫這個 method 時再將已解析好的物件帶入的關鍵。

```php
// vendor/laravel/framework/src/Illuminate/Routing/RouteDependencyResolverTrait.php

protected function resolveClassMethodDependencies(array $parameters, $instance, $method)
    {
        if (! method_exists($instance, $method)) {
            return $parameters;
        }

        return $this->resolveMethodDependencies(
            $parameters, new ReflectionMethod($instance, $method)
        );
    }
```

我們來看看 resolveMethodDependencies 裡面做了什麼：

```php
// vendor/laravel/framework/src/Illuminate/Routing/RouteDependencyResolverTrait.php

public function resolveMethodDependencies(array $parameters, ReflectionFunctionAbstract $reflector)
    {
        $instanceCount = 0;

        $values = array_values($parameters);

        foreach ($reflector->getParameters() as $key => $parameter) {
            $instance = $this->transformDependency(
                $parameter, $parameters
            );

            if (! is_null($instance)) {
                $instanceCount++;

                $this->spliceIntoParameters($parameters, $key, $instance);
            } elseif (! isset($values[$key - $instanceCount]) &&
                      $parameter->isDefaultValueAvailable()) {
                
                $this->spliceIntoParameters($parameters, $key, $parameter->getDefaultValue());
            }
        }

        return $parameters;
    }
```

這邊另外注意，傳入的 parameters 當中可能包含 primitive 值，因為在 Route 當中可能會這樣定義：

```php
// in routes

Route::get('countries.{country}', 'CountryController@show')->name('countries.show');

// in Controllers

public function show($country, Request $request)
{
   ...
}
```

若 endpoint 為 `/api/countries/1` ，我們將 `$parameters` 、 `$reflector->getParameters()` dd 出來看：

```php
// $parameters
array:1 [
  "country" => "1"
]

// $reflector->getParameters()
array:2 [
  0 => ReflectionParameter {#1228
    +name: "country"
    position: 0
  }
  1 => ReflectionParameter {#1229
    +name: "request"
    position: 1
    typeHint: "Illuminate\Http\Request"
  }
]
```

所以在遍歷每個從 ReflectionMethod 得到的 method 引數時 (`$reflector->getParameters()`)，首先會嘗試根據 TypeHint 將其實例化（如果該引數有 typeHint ），若能成功解析出實例，再按照正確位置塞入 $parameters 當中，最後回傳該 $parameters 陣列給vendor/laravel/framework/src/Illuminate/Routing/ControllerDispatcher@dispatch

將這些解析完畢的依賴參數傳入相對應的 Controller method 執行：

```php
$controller->{$method}(...array_values($parameters));
```

至此已經大致將 laravel 實現 Controller  method 自動注入依賴的核心部分都看完了，理解了核心概念再來看這些實作的程式碼後，感覺一切不再那麼 magic 了～

## 參考資料

- Laravel: Up & Running (Book) CHAPTER 11. The The Container
- [https://www.php.net/manual/en/class.reflectionmethod.php](https://www.php.net/manual/en/class.reflectionmethod.php)