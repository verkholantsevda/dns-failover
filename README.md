# DNS Failover

Автоматическое переключение DNS-записей при недоступности основного сервера.

Проект предназначен для отказоустойчивых сервисов, где необходимо автоматически переводить трафик на резервный сервер при сбое основного узла и возвращать обратно после восстановления.

## Возможности

- Проверка доступности серверов через **Prometheus** и **ICMP ping**
- Автоматическое переключение DNS по схеме **A ↔ CNAME**
- Поддержка **нескольких серверов** с циклическим выбором бэкапа
- Интеграция с **Selectel DNS API**
- Уведомления в **Telegram** (с картинками и поддержкой донатов)
- Уведомления в **ntfy.sh** (с краткой информацией о статусе)
- Автоматическое определение страны по коду региона
- Синхронизация состояния DNS при старте сервиса
- Отправка уведомлений **только при смене статуса** (ICMP OK → FAIL и обратно)

## Как работает

Сервис периодически проверяет состояние каждого хоста:

1. **Prometheus** – запрашивается метрика `up{job="...", instance="..."}`.  
   Если ответ успешный (`value == 1`) – хост считается **здоровым**, счётчики сбрасываются, выполняется восстановление A-записи (если сейчас CNAME).

2. Если Prometheus недоступен или вернул `0` – выполняется **ICMP-проверка** основного IP-адреса (из `dns.ip`).

3. При **смене статуса ICMP** (OK → FAIL или FAIL → OK) отправляется уведомление в ntfy (если включён).

4. Если ICMP **FAIL** – увеличивается счётчик ошибок.  
   При достижении порога (`fail_threshold`) выполняется **переключение DNS**:
   - Текущая A-запись удаляется, вместо неё создаётся **CNAME** на следующий хост (циклически).
   - Отправляются уведомления в Telegram (с картинкой) и в ntfy.
   - Счётчики сбрасываются, хост помечается как `Failed`.

5. Если ICMP **OK** – увеличивается счётчик успехов.  
   При достижении порога (`success_threshold`) выполняется **восстановление DNS**:
   - CNAME удаляется, создаётся A-запись с оригинальным IP.
   - Отправляются уведомления о восстановлении.

6. **При старте** сервис проверяет текущую DNS-запись и синхронизирует состояние:
   - Если запись CNAME, а хост здоров – восстанавливает A.
   - Если запись A, а хост недоступен – переключает на CNAME.
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

Пример `configs/config.yaml`

```yaml
interval: 30s
fail_threshold: 3
success_threshold: 3
log:
  level: info
  format: text

selectel:
  account_id: "ACCOUNT_ID"
  project_name: "PROJECT_NAME"
  username: "USERNAME"
  password: "PASSWORD"

telegram:
  enabled: true
  token: "BOT_TOKEN"
  chat_id: 123456789

ntfy:
  enabled: true
  url: https://ntfy.sh
  topic: mytopic
  #token: ""

prometheus:
  url: "http://prometheus:9090"

support:
  enabled: true
  links:
    - title: "Донаты"
      url: "https://example.com/donate"

hosts:
  - name: swe01
    country: SE
    prometheus:
      job: integrations/node_exporter
      instance: swe01.pooler.warp.ignorelist.com
    dns:
      zone: "c26dd837-9bb8-4619-9d24-a615f21bfe09"
      record: "swe01.pooler.warp.ignorelist.com."
      ip: 193.23.200.139

  - name: est011
    country: EE
    prometheus:
      job: integrations/node_exporter
      instance: est01.pooler.warp.ignorelist.com
    dns:
      zone: "c26dd837-9bb8-4619-9d24-a615f21bfe09"
      record: "est01.pooler.warp.ignorelist.com."
      ip: 31.76.55.152
```
## Уровни логирования

| Уровень в config.yaml | DEBUG | INFO | WARN | ERROR |
|-----------------------|:-----:|:----:|:----:|:-----:|
| `debug`               |  ✅   |  ✅  |  ✅  |  ✅   |
| `info`                |  ❌   |  ✅  |  ✅  |  ✅   |
| `warn`                |  ❌   |  ❌  |  ✅  |  ✅   |
| `error`               |  ❌   |  ❌  |  ❌  |  ✅   |

Настройка DNS

* Каждый хост имеет одну A-запись с указанным IP.
* При переключении A-запись заменяется на CNAME, который указывает на следующий хост (циклически).
* Восстановление удаляет CNAME и создаёт A-запись с оригинальным IP.
* Бэкап-хост выбирается автоматически по порядку из списка hosts.

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

## Уведомления
### Telegram
    Отправляются только при переключении DNS (Failover) и восстановлении (Recovery).
    Содержат картинку (failover.jpg или recovery.jpg) и подробный текст с флагами, названиями стран и ссылками на поддержку.
    Формат сообщений дружелюбный для обычных пользователей.

### ntfy.sh
    Отправляются при каждом изменении статуса ICMP (OK → FAIL и FAIL → OK), а также при DNS-переключении и восстановлении.
    Формат сообщений краткий, без изображений:
        ICMP DOWN: 🔴 🇸🇪 host_name (XXX.XXX.XXX.XXX) is unreachable / 16:15
        ICMP UP: 🟢 🇸🇪 host_name  (XXX.XXX.XXX.XXX) is back online / 16:22
        DNS Switch: добавляется строка DNS Switched to 🇪🇪 Estonia
        DNS Restore: добавляется строка DNS Restored

## Синхронизация при старте

При запуске сервис:
    Запрашивает текущую DNS-запись для каждого хоста.
    Проверяет доступность хоста (Prometheus → ICMP).
    Если запись CNAME, а хост здоров – восстанавливает A.
    Если запись A, а хост недоступен – переключает на CNAME.git 
    Все изменения сопровождаются уведомлениями.

Это гарантирует, что после перезапуска сервиса состояние DNS будет соответствовать реальной доступности серверов.

## Структура проекта

```
dns-failover/
│
├── cmd/
│   └── failover/
│       └── main.go              # Точка входа
│
├── configs/
│   └── config.yaml.example      # Пример конфигурации
│
├── internal/
│   ├── checker/                 # Проверки доступности
│   │   ├── icmp.go
│   │   └── metrics.go
│   │
│   ├── config/                  # Загрузка конфигурации
│   │   └── loader.go
│   │
│   ├── dns/                     # DNS-провайдеры
│   │   ├── provider.go          # Интерфейс
│   │   ├── selectel.go          # Реализация для Selectel
│   │   └── selector.go          # Вспомогательные функции
│   │
│   ├── failover/                # Логика переключения
│   │   └── failover.go
│   │
│   ├── notifier/                # Уведомления
│   │   ├── notifier.go          # Интерфейс Notifier
│   │   ├── telegram.go          # Telegram
│   │   ├── ntfy.go              # Ntfy
│   │   ├── multi.go             # Композитный нотификатор
│   │   ├── templates.go         # Шаблоны сообщений Telegram
│   │   ├── flags.go             # Страны и флаги
│   │   ├── countries.go         # Вспомогательные функции для стран
│   │   └── support.go           # Структура поддержки
│   │
│   ├── monitor/                 # Основной цикл мониторинга
│   │   └── monitor.go
│   │
│   ├── logger/                  # Настройка логирования
│   │   └── logger.go
│   │
│   ├── state/                   # Состояние хостов
│   │   └── state.go
│   │
│   └── model/                   # Модели данных
│       ├── types.go
│       └── record.go
│
├── go.mod
└── LICENSE
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