# metrics-overseer

Проект на базе шаблона «Сервер сбора метрик и алертинга».

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Бенчмаркинг и оптимизация кода серверной части

Добавлены бенчмарки для наиболее часто выполняющихся частей кода сервиса

- Вычисление хэша при изменении значения метрики
- Запись метрики в memory-based хранилище (тесты производительности выполняются в этом режиме, т.к. он наименее зависим от ввода/вывода)
- Аудит изменения значения метрик - для записи в файл

Для эмуляции нагрузки на сервис создана вспомогательная программа в каталоге cmd/tools/spammer
Профиль потребления памяти создан для сервиса под тестовой нагрузкой: 10 горутин, отправляющих случайные метрики на сервер батчами по 10000 1 раз в секунду
В результате был оптимизирован код метода updateHash для обновления хэша метрики при сохранении в память

Профили памяти до и после оптимизации:

- profiles/base.pprof
- profiles/result.pprof

Результат сравнения до/после оптимизации:

```
Showing nodes accounting for -616.13MB, 66.30% of 929.32MB total
Dropped 99 nodes (cum <= 4.65MB)
      flat  flat%   sum%        cum   cum%
 -241.02MB 25.93% 25.93%  -242.02MB 26.04%  github.com/andrewsvn/metrics-overseer/internal/model.(*Metrics).UpdateHash
 -193.01MB 20.77% 46.70%  -332.52MB 35.78%  github.com/andrewsvn/metrics-overseer/internal/model.NewCounterMetricsWithDelta (inline)
 -122.60MB 13.19% 59.90%  -557.62MB 60.00%  github.com/andrewsvn/metrics-overseer/internal/repository.(*MemStorage).addCounterInMutex
  -59.50MB  6.40% 66.30%   -59.50MB  6.40%  encoding/json.(*decodeState).literalStore
         0     0% 66.30%   -61.50MB  6.62%  encoding/json.(*decodeState).array
         0     0% 66.30%   -61.50MB  6.62%  encoding/json.(*decodeState).object
         0     0% 66.30%   -61.50MB  6.62%  encoding/json.(*decodeState).unmarshal
         0     0% 66.30%   -61.50MB  6.62%  encoding/json.(*decodeState).value
         0     0% 66.30%   -61.50MB  6.62%  encoding/json.Unmarshal
         0     0% 66.30%  -625.58MB 67.32%  github.com/andrewsvn/metrics-overseer/internal/handler.(*MetricsHandlers).GetRouter.func2.(*MetricsHandlers).updateBatchHandler.1
         0     0% 66.30%  -625.58MB 67.32%  github.com/andrewsvn/metrics-overseer/internal/handler/middleware.(*Authorization).Middleware-fm.(*Authorization).Middleware.func1
         0     0% 66.30%  -625.58MB 67.32%  github.com/andrewsvn/metrics-overseer/internal/handler/middleware.(*Compressing).Middleware-fm.(*Compressing).Middleware.func1
         0     0% 66.30%  -625.58MB 67.32%  github.com/andrewsvn/metrics-overseer/internal/handler/middleware.(*HTTPLogging).Middleware-fm.(*HTTPLogging).Middleware.func1
         0     0% 66.30%  -102.51MB 11.03%  github.com/andrewsvn/metrics-overseer/internal/model.(*Metrics).AddCounter (inline)
         0     0% 66.30%  -558.12MB 60.06%  github.com/andrewsvn/metrics-overseer/internal/repository.(*MemStorage).BatchUpdate
         0     0% 66.30%  -561.10MB 60.38%  github.com/andrewsvn/metrics-overseer/internal/service.(*MetricsService).BatchAccumulateMetrics
         0     0% 66.30%  -625.58MB 67.32%  github.com/go-chi/chi/v5.(*Mux).Mount.func1
         0     0% 66.30%  -625.58MB 67.32%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 66.30%  -625.58MB 67.32%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 66.30%  -625.58MB 67.32%  net/http.(*conn).serve
         0     0% 66.30%  -625.58MB 67.32%  net/http.HandlerFunc.ServeHTTP
         0     0% 66.30%  -625.58MB 67.32%  net/http.serverHandler.ServeHTTP
```
