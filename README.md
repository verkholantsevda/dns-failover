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

├── cmd/
│   └── failover/
│       └── main.go
│
├── configs/
│   └── config.yaml.example
│
├── internal/
│
│   ├── checker/
│   │   ├── icmp.go
│   │   └── metrics.go
│   │
│   ├── config/
│   │   └── loader.go
│   │
│   ├── dns/
│   │   ├── provider.go
│   │   └── selectel.go
│   │
│   ├── failover/
│   │   └── failover.go
│   │
│   ├── notifier/
│   │   ├── telegram.go
│   │   └── flags.go
│   │
│   ├── monitor/
│   │   └── monitor.go
│   │
│   └── state/
│       └── state.go
│
└── go.mod
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