# DNS Failover

Автоматическое переключение DNS-записей при недоступности основного сервера.

Проект предназначен для отказоустойчивых сервисов, где необходимо автоматически переводить трафик на резервный сервер при сбое основного узла и возвращать обратно после восстановления.

## Возможности

- Проверка доступности серверов
- Проверка через Prometheus
- Дополнительная проверка через ICMP ping
- Автоматическое переключение DNS
- Поддержка нескольких серверов с приоритетами
- Поддержка Selectel DNS API
- Уведомления в Telegram
- Автоматическое определение страны через код региона
- Ведение состояния текущего активного сервера

## Как работает

Сервис периодически проверяет состояние серверов:

1. Выполняется проверка через Prometheus.
2. Если сервер недоступен — выполняется дополнительная ICMP-проверка.
3. После достижения заданного количества ошибок:
   - основной сервер отключается в DNS;
   - резервный сервер включается;
   - отправляется уведомление в Telegram.
4. После восстановления основного сервера:
   - DNS автоматически возвращается обратно;
   - отправляется уведомление.

Пример уведомления:

```
🇸🇪 WARP Sweden

Из-за недоступности сервера

трафик временно переключен на 🇫🇷 France.

Соединение продолжает работать в штатном режиме.

Мы автоматически вернем маршрут после восстановления сервера.
```

## Архитектура

```
                +----------------+
                |   Prometheus   |
                +-------+--------+
                        |
                        |
+-------------+    +----v-----+
|  Servers    |--->| Monitor  |
+-------------+    +----+-----+
                        |
                        |
              +---------+----------+
              |                    |
       +------v------+      +------v------+
       | Selectel    |      | Telegram    |
       | DNS API     |      | Bot API     |
       +-------------+      +-------------+
```

## Требования

- Go 1.24+
- Доступ к DNS API провайдера
- Telegram Bot Token (опционально)
- Prometheus (опционально)

## Установка

Клонировать репозиторий:

```bash
git clone https://github.com/verkholantsevda//dns-failover.git

cd dns-failover
```

Создать конфигурацию:

```bash
cp configs/config.yaml.example configs/config.yaml
```

Отредактировать:

```bash
nano configs/config.yaml
```

Собрать проект:

```bash
go build -o dns-failover ./cmd/failover
```

Запуск:

```bash
./dns-failover
```

## Конфигурация

Пример:

```yaml
interval: 30s

log:
  level: info
  format: text  #json/text

fail_threshold: 3

success_threshold: 3


selectel:
  account_id: "ACCOUNT_ID"
  project_name: "PROJECT_NAME"
  username: "USERNAME"
  password: "PASSWORD"


telegram:
  token: "BOT_TOKEN"
  chat_id: 123456789

support:
  enabled: true
  links:
    - title: "Донаты"
      url: "https://example.com/donate"
    - title: "Поддержка проекта"
      url: "https://example.com/support"

prometheus:
  url: "http://prometheus:9090"


hosts:

  - name: example-service

    icmp_host: example.com

    prometheus:
      job: node
      instance: example.com:9100


    dns:
      zone: "ZONE_ID"

      record:
        "service.example.com."


      records:

        - ip: 192.0.2.10
          country: US
          priority: 1
          disabled: false


        - ip: 192.0.2.20
          country: DE
          priority: 2
          disabled: false
```
## Уровни логирования

| Уровень в config.yaml | DEBUG | INFO | WARN | ERROR |
|-----------------------|:-----:|:----:|:----:|:-----:|
| `debug`               |  ✅   |  ✅  |  ✅  |  ✅   |
| `info`                |  ❌   |  ✅  |  ✅  |  ✅   |
| `warn`                |  ❌   |  ❌  |  ✅  |  ✅   |
| `error`               |  ❌   |  ❌  |  ❌  |  ✅   |

## Настройка DNS

DNS-запись должна содержать несколько A-records.

Пример:

```
service.example.com

A 192.0.2.10
A 192.0.2.20
```

Переключение выполняется через параметр:

```json
{
  "disabled": true
}
```

Неактивная запись исключается из ответа DNS.

## Telegram уведомления

Для получения уведомлений:

1. Создать Telegram Bot через BotFather.
2. Получить token.
3. Узнать chat_id пользователя или группы.
4. Добавить параметры:

```yaml
telegram:
  token: "TOKEN"
  chat_id: ID
```

## Получение Telegram Chat ID

Отправить сообщение своему боту.

Затем выполнить:

```bash
curl https://api.telegram.org/bot<TOKEN>/getUpdates
```

В ответе будет:

```json
{
  "message": {
    "chat": {
      "id": 123456789
    }
  }
}
```

Это значение используется в конфигурации.

## Структура проекта

```
dns-failover/
│
├── cmd/                         # Точки входа приложения (исполняемые программы)
│   └── failover/
│       └── main.go              # Запуск сервиса: загрузка конфига, инициализация компонентов, старт мониторинга
│
├── configs/                     # Конфигурационные файлы
│   └── config.yaml.example      # Пример конфигурации: DNS, Telegram, Prometheus, логирование
│
├── internal/                    # Внутренние пакеты приложения (не доступны извне)
│
│   ├── checker/                 # Проверки доступности серверов
│   │   ├── icmp.go              # ICMP/Ping проверка доступности узла
│   │   └── metrics.go           # Проверка состояния через Prometheus API
│
│   ├── config/                  # Работа с конфигурацией
│   │   └── loader.go            # Загрузка и разбор config.yaml
│
│   ├── dns/                     # Работа с DNS-провайдерами
│   │   ├── provider.go          # Интерфейс DNS-провайдера
│   │   └── selectel.go          # Реализация управления DNS Selectel API
│
│   ├── failover/                # Логика переключения DNS
│   │   └── failover.go          # Отключение основного IP, включение резервного, уведомления
│
│   ├── notifier/                # Уведомления
│   │   ├── telegram.go          # Отправка сообщений и изображений через Telegram Bot API
│   │   └── flags.go             # Список стран, флаги и названия регионов
│
│   ├── monitor/                 # Основной цикл мониторинга
│   │   └── monitor.go           # Проверка серверов, обработка состояний, запуск failover/recovery
│
│   ├── logger/                  # Система логирования
│   │   └── logger.go            # Настройка slog: уровень (debug/info/warn/error) и формат (text/json)
│
│   └── state/                   # Хранение состояния
│       └── state.go             # Состояние хоста: активный IP, количество успешных/неуспешных проверок
│
└── go.mod                       # Описание Go-модуля и зависимостей
```

## Безопасность

Не храните реальные данные:

- API пароли
- Telegram token
- реальные IP
- внутренние домены

в репозитории.

Используйте:

- `configs/config.yaml`
- переменные окружения
- секреты CI/CD


`configs/config.yaml` добавлен в `.gitignore`.

## Лицензия
MIT License. Подробности см. в файле [LICENSE](./LICENSE).