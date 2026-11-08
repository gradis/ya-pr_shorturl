# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
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

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**


## Бенчмарки и профилирование памяти

Замеры выполнены 27.10.2026: Go 1.27.1, macOS darwin/arm64, Apple M3 Pro.
Бенчмарки измеряют генерацию ID, сокращение существующего URL, пакет из 10 URL,
переход по ссылке, получение 100 ссылок пользователя, операции memory repository
и HTTP-хэндлер перехода через Gin. Все бенчмарки проверяют ошибки и используют
непустые данные; в HTTP-бенчмарке измеряются также recorder и запись ответа.

```sh
go test ./... -run '^$' -bench . -benchmem -benchtime=100ms
```

### Воспроизводимая нагрузка

`BenchmarkProfileWorkload` подготавливает 1000 оригинальных URL со строками запросов:
100 принадлежат текущему пользователю, 900 — другому. Одна итерация выполняет
повторное сокращение, batch из 10 уже известных URL, переход и получение списка
из 100 ссылок. Повторные URL позволяют держать размер хранилища постоянным:
сравниваются одни и те же 1000 итераций, а не разные объёмы накопленных записей.
Это нагрузка на сервис и memory repository внутри процесса, без сети и PostgreSQL;
она не измеряет производительность внешней БД, диска или удалённого аудита.

Исходная реализация — коммит `be08668`, к которому до оптимизации добавлены
те же файлы бенчмарков. Профиль `base.pprof` снят **до** изменения рабочего кода,
`result.pprof` — после, с одинаковыми параметрами:

```sh
# На исходной реализации с добавленными бенчмарками:
go test ./internal/service -run '^$' -bench '^BenchmarkProfileWorkload$' \
  -benchtime=1000x -benchmem -memprofilerate=1 \
  -memprofile=profiles/base.pprof -o /tmp/shorturl-iter17-base.test

# После оптимизации:
go test ./internal/service -run '^$' -bench '^BenchmarkProfileWorkload$' \
  -benchtime=1000x -benchmem -memprofilerate=1 \
  -memprofile=profiles/result.pprof -o /tmp/shorturl-iter17-result.test
```

`memprofilerate=1` учитывает каждую аллокацию, но существенно замедляет выполнение.
Поэтому время ниже измерено отдельно, без профилировщика: каждая версия запускалась
три раза с одинаковой командой (меняется только имя бинарного файла):

```sh
/tmp/shorturl-iter17-base.test -test.run '^$' \
  -test.bench 'BenchmarkGenerateID|BenchmarkURLService|BenchmarkProfileWorkload' \
  -test.benchtime=1000x -test.benchmem -test.count=3
```

Приведены медианы; полные результаты сохранены в
[base-bench.txt](profiles/base-bench.txt) и [result-bench.txt](profiles/result-bench.txt).
Время коротких замеров зависит от фоновой нагрузки; B/op и allocs/op показывают
более устойчивое сравнение.

| Операция | ns/op до → после | B/op до → после | allocs/op до → после |
| --- | ---: | ---: | ---: |
| Генерация ID | 1021 → 317 | 392 → 8 | 25 → 1 |
| Повторное сокращение | 1297 → 563 | 600 → 184 | 29 → 3 |
| Batch из 10 | 11442 → 5055 | 8116 → 3312 | 323 → 33 |
| Список из 100 | 14482 → 17169 | 26080 → 12800 | 309 → 102 |
| Смешанная нагрузка | 26289 → 21696 | 34797 → 16296 | 661 → 138 |

### Анализ pprof и оптимизация

Использованы `top`, `list`, `peek`, `web`; `go tool pprof` — поставляемая с Go
версия pprof. Для `web` необходим установленный Graphviz (`dot`).

```sh
go tool pprof -top -alloc_space profiles/base.pprof
go tool pprof -top -alloc_objects profiles/base.pprof
go tool pprof -top -inuse_space profiles/base.pprof
go tool pprof -list 'generateID|GetUserURLs|AddBatchURLs' -alloc_space profiles/base.pprof
go tool pprof -peek generateID -alloc_objects profiles/base.pprof
go tool pprof -web -alloc_space profiles/base.pprof
```

В исходном профиле `MemoryRepository.GetByUserID` занимает 45,67% alloc_space
(15,46 МиБ): постепенное расширение среза выделяло несколько промежуточных массивов.
Теперь записи подсчитываются под тем же RLock, затем выделяется срез точного размера.
Аналогичное изменение внесено в файловое хранилище. Цена — дополнительный проход
по map: список в этом замере стал медленнее примерно на 19%, но объём аллокаций
уменьшился вдвое, а смешанная нагрузка в целом ускорилась.

`GetUserURLs` вместе с вызываемыми функциями занимает 73,49% alloc_space.
Простое соединение строк заменило `fmt.Sprintf` при сборке короткого URL.
В batch удалено создание предварительного ShortURL, который сразу перезаписывался
после сохранения: ответ формируется по ID, реально возвращённым репозиторием.

`peek` и `list` выявили избыточные аллокации в ветке `generateID`,
включая `crypto/rand.Int` и `math/big`. Генератор теперь читает случайные байты
блоком в стековый буфер; значения 248–255 отбрасываются, чтобы остаток по модулю 62
не создавал смещения. Сохранены криптографический источник, длина и алфавит ID.

Визуальный граф показывает основные ветки GetUserURLs → GetByUserID / fmt.Sprintf
и AddBatchURLs → generateID → crypto/rand.Int. Граф разницы сохранён в
[allocations.svg](profiles/allocations.svg), исходные отчёты — в
[base-top.txt](profiles/base-top.txt), [base-list.txt](profiles/base-list.txt),
[base-peek.txt](profiles/base-peek.txt).

Отдельно изучена удерживаемая память (`inuse_space`): около 46,5 КиБ до и 41,5 КиБ
после, преимущественно runtime и инфраструктура теста. По завершении бенчмарка
тестовое хранилище уже освобождено, поэтому этот профиль не показывает размер
живого хранилища работающего сервера. Выигрыш в удерживаемой памяти сервера не
заявляется; основное измеренное улучшение относится к суммарным аллокациям.
Отчёты: [base-inuse.txt](profiles/base-inuse.txt),
[result-inuse.txt](profiles/result-inuse.txt), [diff-inuse.txt](profiles/diff-inuse.txt).

### Сравнение профилей

Выполнена команда из задания через встроенный в Go pprof:

```sh
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Полученный вывод (в этой версии pprof по умолчанию выбран `alloc_space`):

```text
File: service.test
Type: alloc_space
Time: 2026-10-27 20:23:42 +03
Showing nodes accounting for -18046.97kB, 52.07% of 34655.87kB total
Dropped 80 nodes (cum <= 173.28kB)
      flat  flat%   sum%        cum   cum%
-9828.12kB 28.36% 28.36% -9828.12kB 28.36%  github.com/gradis/ya-pr_shorturl/internal/repository.(*MemoryRepository).GetByUserID
-3781.36kB 10.91% 39.27% -3797.65kB 10.96%  fmt.Sprintf
-3437.36kB  9.92% 49.19% -4125.16kB 11.90%  crypto/rand.Int
 -687.67kB  1.98% 51.17%  -687.67kB  1.98%  math/big.nat.make (inline)
 -312.50kB   0.9% 52.07% -4687.45kB 13.53%  github.com/gradis/ya-pr_shorturl/internal/service.(*URLService).AddBatchURLs
    0.05kB 0.00014% 52.07% -4124.94kB 11.90%  github.com/gradis/ya-pr_shorturl/internal/service.generateID
         0     0% 52.07%  -406.23kB  1.17%  github.com/gradis/ya-pr_shorturl/internal/service.(*URLService).AddURL
         0     0% 52.07% -12969.52kB 37.42%  github.com/gradis/ya-pr_shorturl/internal/service.(*URLService).GetUserURLs
         0     0% 52.07%  -374.98kB  1.08%  github.com/gradis/ya-pr_shorturl/internal/service.(*URLService).saveWithUniqueID
         0     0% 52.07% -18063.21kB 52.12%  github.com/gradis/ya-pr_shorturl/internal/service.BenchmarkProfileWorkload
         0     0% 52.07%  -687.67kB  1.98%  math/big.(*Int).SetUint64 (inline)
         0     0% 52.07%  -687.67kB  1.98%  math/big.nat.setUint64
         0     0% 52.07%  -687.67kB  1.98%  math/big.nat.setWord (inline)
         0     0% 52.07% -18063.10kB 52.12%  testing.(*B).run1.func1
         0     0% 52.07% -18061.44kB 52.12%  testing.(*B).runN
```

Отрицательные значения означают сокращение аллокаций в соответствующих функциях.
Суммарное снижение alloc_space профиля — около 52%; в отдельном бенчмарке смешанной
нагрузки B/op уменьшилось примерно на 53%, число аллокаций — на 79%.
Для явного выбора метрики независимо от версии pprof:

```sh
go tool pprof -top -alloc_space -diff_base=profiles/base.pprof profiles/result.pprof
go tool pprof -top -alloc_objects -diff_base=profiles/base.pprof profiles/result.pprof
go tool pprof -top -inuse_space -diff_base=profiles/base.pprof profiles/result.pprof
```

### Проверка тестов и покрытия

```sh
go test -race ./... -coverpkg=./... -coverprofile=/tmp/shorturl-iter17-coverage.out
go tool cover -func=/tmp/shorturl-iter17-coverage.out
go vet ./...
```

Итог: **63,2% statements** по всему модулю (`-coverpkg=./...`), включая пакеты
без собственных тестов; требование 40% выполнено. Все тесты прошли с `-race`,
`go vet ./...` также завершился успешно. Полный отчёт —
[coverage.txt](profiles/coverage.txt).

Восстановлены регрессионные проверки длины и алфавита ID, batch с новыми и
существующими ссылками, фильтрации ссылок пользователя, независимости возвращаемых
записей и отмены контекста для обоих локальных хранилищ.

## Godoc и примеры эндпоинтов

Основные экспортированные типы, интерфейсы, функции, методы и ошибки описаны
в godoc-комментариях; общие сведения о пакетах находятся в `doc.go`.
Просмотр документации без дополнительных инструментов:

```sh
go doc ./internal/handler URLHandler.RegisterRoutes
go doc -all ./internal/service
go doc -all ./internal/repository
```

В `internal/handler/example_test.go` находятся семь исполняемых примеров:
сокращение текстового и JSON URL, переход, пакетное сокращение, список ссылок
пользователя, асинхронное удаление и проверка `/ping`. Примеры используют настоящий
роутер Gin, сервис, хранилище в памяти и подписанную cookie. Ожидаемый вывод
проверяется стандартными комментариями `Output:`; внешняя БД и запущенный сервер
не требуются.

```sh
go test ./internal/handler -run '^Example' -v
```

Если установлен `godoc`, запустите `godoc -http=localhost:6060` и откройте
`http://localhost:6060/pkg/github.com/gradis/ya-pr_shorturl/internal/handler/?m=all`.
Параметр `m=all` позволяет просматривать пакет из каталога `internal`.

После добавления примеров общее покрытие модуля составляет **64,4% statements**
(проверено с `-coverpkg=./...`). Полный набор тестов с `-race` и `go vet ./...`
проходит; примеры входят в обычный запуск `go test ./...`.
