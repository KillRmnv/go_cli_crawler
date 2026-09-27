# go_cli_crawler

CLI-краулер: обходит ссылки с заданных URL вглубь (с ограничением по глубине и общему
таймауту) и сохраняет результат в виде JSON-дерева.

## Сборка

```bash
go build ./...        
go build -o crawler ./cmd  
```

## Запуск

```bash
./crawler \
  -urls https://example.com,https://google.com \
  -depth 4 \
  -timeout 2m \
  -request-timeout 10s \
  -retry 1 \
  -delay 0s \
  -output ./out/result.json \
  -log ./log/crawler.log
```

```bash
go run ./cmd -urls https://example.com,https://google.com
```

## Флаги

| Флаг               | По умолчанию        | Описание |
|--------------------|---------------------|----------|
| `-urls`            | *(обязательный)*    | URL через запятую — корни обхода. Без них в лог пишется `Invalid url`, результат будет пустым. |
| `-depth`           | `10`                | Максимальная глубина обхода (семя = 1). |
| `-timeout`         | `2m`                | Общий лимит прогона. По истечении обход разворачивается, а всё построенное сохраняется в результат. |
| `-request-timeout` | `10s`               | Таймаут одного HTTP-запроса (HEAD + GET). **Не ставьте `0`** — запросы перестанут ограничиваться и развертывание может зависнуть. |
| `-output`          | `./out/result.json` | Куда писать результат (JSON). |
| `-log`             | `./log/crawler.log` | Куда писать лог; рядом создаётся файл ошибок `*_error.log`. |
| `-retry`           | `1`                 | Общее число попыток запроса на URL (`0` — запросы не выполняются, `1` — одна попытка). |
| `-delay`           | `0s`                | Пауза перед повторной попыткой после ошибки. |
| `-stubs`           | `false`              | Заготовки-стабы в `links` (не загруженные href-ы с пустым `title`). По умолчанию в дереве только реально загруженные страницы; `-stubs=true` — плюс стабы, JSON больше. |

## Выходные файлы

- **Результат** — JSON-массив корневых узлов:
  `{"resource": "<URL страницы>", "title": "<заголовок страницы>", "links": [...]}`,
  где `links` — дочерние страницы, а при `-stubs=true` плюс заготовки
  необработанных ссылок с пустым `title`; по умолчанию (`-stubs=false`) —
  только реально загруженные поддеревья.
- **Не перезапись**: если путь `-output` (или `-log`) уже существует, файл создаётся
  рядом с именем вида `result2026-09-27_15_23.json` / `crawler2026-09-27_15_23.log`.
- Страницы с не-UTF-8 кодировкой попадают в JSON.

## Тесты

Офлайн-тесты (сеть не нужна):

```bash
go test ./internal -run 'TestCrawlerConfig_Setters|TestExtractErrStatusLogFilepath|TestCliCrawler_Init|TestCliCrawler_CreateFile|TestCliCrawler_ParsePage|TestCliCrawler_Crawle|TestCrawle|TestSendNode'
```

Полный `go test ./...` выполняет живые HTTP-запросы (google.com, httpbin.org и т.п.)
и падает без интернета.
